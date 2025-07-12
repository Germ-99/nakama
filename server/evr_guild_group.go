package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type GuildGroup struct {
	GroupMetadata
	State *GuildGroupState
	Group *api.Group
}

func NewGuildGroup(group *api.Group, state *GuildGroupState) (*GuildGroup, error) {

	md := &GroupMetadata{}
	if err := json.Unmarshal([]byte(group.Metadata), md); err != nil {
		return nil, err
	}

	// Ensure the matchmaking channel IDs have been initialized
	if md.MatchmakingChannelIDs == nil {
		md.MatchmakingChannelIDs = make(map[string]string)
	}

	return &GuildGroup{
		GroupMetadata: *md,
		State:         state,
		Group:         group,
	}, nil
}

func (g *GuildGroup) Name() string {
	return g.Group.Name
}

func (g *GuildGroup) Description() string {
	return g.Group.Description
}

func (g *GuildGroup) ID() uuid.UUID {
	return uuid.FromStringOrNil(g.Group.Id)
}

func (g *GuildGroup) IDStr() string {
	return g.Group.Id
}

func (g *GuildGroup) Size() int {
	return int(g.Group.EdgeCount)
}

func (g GuildGroup) MembershipBitSet(userID string) uint64 {
	return guildGroupPermissions{
		IsAllowedMatchmaking: g.IsAllowedMatchmaking(userID),
		IsEnforcer:           g.IsEnforcer(userID),
		IsAuditor:            g.IsAuditor(userID),
		IsServerHost:         g.IsServerHost(userID),
		IsAllocator:          g.IsAllocator(userID),
		IsSuspended:          g.IsSuspended(userID, nil),
		IsLimitedAccess:      g.IsLimitedAccess(userID),
		IsAPIAccess:          g.IsAPIAccess(userID),
		IsAccountAgeBypass:   g.IsAccountAgeBypass(userID),
		IsVPNBypass:          g.IsVPNBypass(userID),
	}.ToUint64()
}

func (g GuildGroup) HasRole(userID, role string) bool {
	g.State.mu.RLock()
	defer g.State.mu.RUnlock()
	return g.State.hasRole(userID, role)
}

func (g *GuildGroup) RoleCacheUpdate(account *EVRProfile, roles []string) bool {
	g.State.mu.Lock()
	defer g.State.mu.Unlock()

	g.State.updated = false
	accountID := account.ID()

	// Initialize the role cache if it's nil.
	if g.State.RoleCache == nil {
		g.State.RoleCache = make(map[string]map[string]struct{})
		g.State.updated = true // Initialization counts as an update
	}

	// Filter out roles that are not relevant to this guild group.
	relevantRoles := make(map[string]struct{})
	for _, role := range roles {
		if _, ok := g.RoleMap.AsSet()[role]; ok {
			relevantRoles[role] = struct{}{}
		}
	}

	// Iterate over all known roles in this guild group to update affiliations.
	for _, roleID := range g.RoleMap.AsSlice() {
		_, accountHasRole := relevantRoles[roleID]
		userIDsInRole, cacheHasRole := g.State.RoleCache[roleID]

		if accountHasRole {
			// Account should have this role.
			if !cacheHasRole {
				// Role not in cache, add it and the account.
				g.State.RoleCache[roleID] = map[string]struct{}{accountID: {}}
				g.State.updated = true
			} else if _, ok := userIDsInRole[accountID]; !ok {
				// Role in cache, but account not associated, add account.
				userIDsInRole[accountID] = struct{}{}
				g.State.updated = true
			}
		} else {
			// Account should not have this role.
			if cacheHasRole {
				if _, ok := userIDsInRole[accountID]; ok {
					// Account is associated with role, but shouldn't be, remove.
					delete(userIDsInRole, accountID)
					g.State.updated = true
					// If no more users in this role, consider cleaning up the role entry (optional).
					if len(userIDsInRole) == 0 {
						delete(g.State.RoleCache, roleID)
					}
				}
			}
		}
	}

	g.updateSuspensionStatus(account, accountID)

	return g.State.updated
}

// updateSuspensionStatus manages the suspension status of an account based on its roles.
func (g *GuildGroup) updateSuspensionStatus(account *EVRProfile, accountID string) {
	isSuspended := g.State.hasRole(accountID, g.RoleMap.Suspended)

	if isSuspended {
		// If the user is suspended, add their XPIDs to the suspension list.
		if g.State.SuspendedXPIDs == nil {
			g.State.SuspendedXPIDs = make(map[evr.EvrId]string)
			g.State.updated = true // Initialization counts as an update
		}
		for _, xpid := range account.XPIDs() {
			if _, ok := g.State.SuspendedXPIDs[xpid]; !ok {
				g.State.SuspendedXPIDs[xpid] = accountID
				g.State.updated = true
			}
		}
	} else {
		// If the user is no longer suspended, remove their XPIDs from the suspension list.
		if g.State.SuspendedXPIDs != nil {
			for _, xpid := range account.XPIDs() {
				if _, ok := g.State.SuspendedXPIDs[xpid]; ok {
					delete(g.State.SuspendedXPIDs, xpid)
					g.State.updated = true
				}
			}
		}
	}
}

func (g *GuildGroup) IsOwner(userID string) bool {
	return g.OwnerID == userID
}

func (g *GuildGroup) IsServerHost(userID string) bool {
	return g.HasRole(userID, g.RoleMap.ServerHost)
}

func (g *GuildGroup) IsAllocator(userID string) bool {
	return g.HasRole(userID, g.RoleMap.Allocator)
}

func (g *GuildGroup) IsAuditor(userID string) bool {
	if slices.Contains(g.NegatedEnforcerIDs, userID) {
		return false
	}
	return g.HasRole(userID, g.RoleMap.Auditor)
}

func (g *GuildGroup) IsEnforcer(userID string) bool {
	if slices.Contains(g.NegatedEnforcerIDs, userID) {
		return false
	}
	return g.HasRole(userID, g.RoleMap.Enforcer)
}

func (g *GuildGroup) IsMember(userID string) bool {
	return g.HasRole(userID, g.RoleMap.Member)
}

func (g *GuildGroup) IsSuspended(userID string, xpid *evr.EvrId) bool {
	g.State.mu.RLock()
	defer g.State.mu.RUnlock()

	if g.State.hasRole(userID, g.RoleMap.Suspended) {
		return true
	}
	if xpid == nil || g.State.SuspendedXPIDs == nil {
		return false
	}

	if _, ok := g.State.SuspendedXPIDs[*xpid]; ok {
		// Check if the user is (still) suspended
		if g.State.hasRole(userID, g.RoleMap.Suspended) {
			return true
		}
	}

	return false
}

func (g *GuildGroup) IsLimitedAccess(userID string) bool {
	return g.HasRole(userID, g.RoleMap.LimitedAccess)
}

func (g *GuildGroup) IsAPIAccess(userID string) bool {
	return g.HasRole(userID, g.RoleMap.APIAccess)
}

func (g *GuildGroup) IsAccountAgeBypass(userID string) bool {
	return g.HasRole(userID, g.RoleMap.AccountAgeBypass)
}

func (g *GuildGroup) IsVPNBypass(userID string) bool {
	return g.HasRole(userID, g.RoleMap.VPNBypass)
}

func (g *GuildGroup) IsAllowedFeature(feature string) bool {
	return slices.Contains(g.AllowedFeatures, feature)
}

func (g *GuildGroup) IsAllowedMatchmaking(userID string) bool {
	if !g.MembersOnlyMatchmaking {
		return true
	}
	g.State.mu.RLock()
	defer g.State.mu.RUnlock()

	if g.State.RoleCache == nil {
		return false
	}

	if userIDs, ok := g.State.RoleCache[g.RoleMap.Member]; ok {
		if _, ok := userIDs[userID]; ok {
			return true
		}
	}

	return false
}

func GuildGroupLoad(ctx context.Context, nk runtime.NakamaModule, groupID string) (*GuildGroup, error) {
	groups, err := nk.GroupsGetId(ctx, []string{groupID})
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %v", err)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("group not found")
	}

	state, err := GuildGroupStateLoad(ctx, nk, ServiceSettings().DiscordBotUserID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guild group state: %v", err)
	}

	return NewGuildGroup(groups[0], state)
}

func GuildGroupStore(ctx context.Context, nk runtime.NakamaModule, guildGroupRegistry *GuildGroupRegistry, group *GuildGroup) error {
	_nk, ok := nk.(*RuntimeGoNakamaModule)
	if !ok {
		return fmt.Errorf("failed to cast nakama module")
	}

	// Store the State
	err := StorageWrite(ctx, nk, ServiceSettings().DiscordBotUserID, group.State)
	if err != nil {
		return fmt.Errorf("failed to write guild group state: %v", err)
	}

	// Store the metadata
	if err := GroupMetadataSave(ctx, _nk.db, group.Group.Id, &group.GroupMetadata); err != nil {
		return fmt.Errorf("failed to save guild group metadata: %v", err)
	}
	if guildGroupRegistry != nil {
		guildGroupRegistry.Add(group)
	}
	return nil
}

func GuildUserGroupsList(ctx context.Context, nk runtime.NakamaModule, guildGroupRegistry *GuildGroupRegistry, userID string) (map[string]*GuildGroup, error) {
	guildGroups := make(map[string]*GuildGroup, 0)
	cursor := ""
	for {
		// Fetch the groups using the provided userId
		groups, _, err := nk.UserGroupsList(ctx, userID, 100, nil, cursor)
		if err != nil {
			return nil, fmt.Errorf("error getting user groups: %w", err)
		}
		for _, ug := range groups {
			if ug.State.Value > int32(api.UserGroupList_UserGroup_MEMBER) {
				continue
			}
			switch ug.Group.GetLangTag() {
			case "guild":

				if guildGroupRegistry != nil {
					if gg := guildGroupRegistry.Get(ug.Group.Id); gg != nil {
						guildGroups[ug.Group.Id] = gg
					}
				} else {
					group, err := GuildGroupLoad(ctx, nk, ug.Group.Id)
					if err != nil {
						return nil, fmt.Errorf("error loading guild group: %w", err)
					}
					guildGroups[ug.Group.Id] = group
				}
			}
		}
		if cursor == "" {
			break
		}
	}
	return guildGroups, nil
}

func CreateGuildGroup(ctx context.Context, logger *zap.Logger, db *sql.DB, guildID string, userID uuid.UUID, creatorID uuid.UUID, name, lang, desc, avatarURL, metadata string, open bool, maxCount int) (*api.Group, error) {
	if userID == uuid.Nil {
		return nil, runtime.ErrGroupCreatorInvalid
	}

	state := 1
	if open {
		state = 0
	}

	params := []interface{}{SnowflakeToUUID(guildID), creatorID, name, desc, avatarURL, state}
	statements := []string{"$1", "$2", "$3", "$4", "$5", "$6"}

	query := "INSERT INTO groups(id, creator_id, name, description, avatar_url, state"

	// Add lang tag if any.
	if lang != "" {
		query += ", lang_tag"
		params = append(params, lang)
		statements = append(statements, "$"+strconv.Itoa(len(params)))
	}
	// Add max count if any.
	if maxCount > 0 {
		query += ", max_count"
		params = append(params, maxCount)
		statements = append(statements, "$"+strconv.Itoa(len(params)))
	}
	// Add metadata if any.
	if metadata != "" {
		query += ", metadata"
		params = append(params, metadata)
		statements = append(statements, "$"+strconv.Itoa(len(params)))
	}

	// Add the trailing edge count value.
	query += `, edge_count) VALUES (` + strings.Join(statements, ",") + `,1)
RETURNING id, creator_id, name, description, avatar_url, state, edge_count, lang_tag, max_count, metadata, create_time, update_time`

	var group *api.Group

	if err := ExecuteInTx(ctx, db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, params...)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == dbErrorUniqueViolation {
				logger.Info("Could not create group as it already exists.", zap.String("name", name))
				return runtime.ErrGroupNameInUse
			}
			logger.Debug("Could not create group.", zap.Error(err))
			return err
		}
		// Rows closed in groupConvertRows()

		groups, _, err := groupConvertRows(rows, 1)
		if err != nil {
			logger.Debug("Could not parse rows.", zap.Error(err))
			return err
		}

		group = groups[0]
		_, err = groupAddUser(ctx, db, tx, uuid.Must(uuid.FromString(group.Id)), userID, 0)
		if err != nil {
			logger.Debug("Could not add user to group.", zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		if errors.Is(err, runtime.ErrGroupNameInUse) {
			return nil, runtime.ErrGroupNameInUse
		}
		logger.Error("Error creating group.", zap.Error(err))
		return nil, err
	}

	logger.Info("Group created.", zap.String("group_id", group.Id), zap.String("user_id", userID.String()))

	return group, nil
}
