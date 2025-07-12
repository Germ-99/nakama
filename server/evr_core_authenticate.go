package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	SystemUserID = "00000000-0000-0000-0000-000000000000"

	ActivePartyGroupIndex     = "Index_PartyGroup"
	CacheStorageCollection    = "Cache"
	IPinfoCacheKey            = "IPinfo"
	CosmeticLoadoutCollection = "CosmeticLoadouts"
	CosmeticLoadoutKey        = "loadouts"
	VRMLStorageCollection     = "VRML"

	// The Application IDs for the default clients
	NoOvrAppId uint64 = 0x0
	QuestAppId uint64 = 0x7de88f07bd07a
	PcvrAppId  uint64 = 0x4dd2b684a47fa
)

func AuthenticateDiscord(ctx context.Context, logger *zap.Logger, db *sql.DB, discordID string, create bool) (string, string, bool, error) {
	found := true

	// Look for an existing account.
	query := "SELECT id, username, disable_time FROM users WHERE custom_id = $1"
	var dbUserID string
	var dbUsername string
	var dbDisableTime pgtype.Timestamptz
	err := db.QueryRowContext(ctx, query, discordID).Scan(&dbUserID, &dbUsername, &dbDisableTime)
	if err != nil {
		if err == sql.ErrNoRows {
			found = false
		} else {
			logger.Error("Error looking up user by discord ID.", zap.Error(err), zap.String("discordID", discordID), zap.Bool("create", create))
			return "", "", false, status.Error(codes.Internal, "Error finding user account.")
		}
	}

	// Existing account found.
	if found {
		// Check if it's disabled.
		if dbDisableTime.Valid && dbDisableTime.Time.Unix() != 0 {
			logger.Info("User account is disabled.", zap.String("discordID", discordID), zap.Bool("create", create))
			return "", "", false, status.Error(codes.PermissionDenied, "User account banned.")
		}

		return dbUserID, dbUsername, false, nil
	}

	if !create {
		// No user account found, and creation is not allowed.
		return "", "", false, status.Error(codes.NotFound, "User account not found.")
	}
	username := generateUsername()
	// Create a new account.
	userID := SnowflakeToUUID(discordID).String()
	if userID == "" || userID == uuid.Nil.String() {
		logger.Error("Invalid Discord ID provided for user creation.", zap.String("discordID", discordID), zap.String("username", username), zap.Bool("create", create))
		return "", "", false, status.Error(codes.InvalidArgument, "Invalid Discord ID provided.")
	}

	query = "INSERT INTO users (id, username, custom_id, create_time, update_time) VALUES ($1, $2, $3, now(), now())"
	result, err := db.ExecContext(ctx, query, userID, username, discordID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == dbErrorUniqueViolation {
			if strings.Contains(pgErr.Message, "users_username_key") {
				// Username is already in use by a different account.
				return "", "", false, status.Error(codes.AlreadyExists, "Username is already in use.")
			} else if strings.Contains(pgErr.Message, "users_custom_id_key") {
				// A concurrent write has inserted this discord ID.
				logger.Info("Did not insert new user as discord ID already exists.", zap.Error(err), zap.String("discordID", discordID), zap.String("username", username), zap.Bool("create", create))
				return "", "", false, status.Error(codes.Internal, "Error finding or creating user account.")
			}
		}
		logger.Error("Cannot find or create user with discord ID.", zap.Error(err), zap.String("discordID", discordID), zap.String("username", username), zap.Bool("create", create))
		return "", "", false, status.Error(codes.Internal, "Error finding or creating user account.")
	}

	if rowsAffectedCount, _ := result.RowsAffected(); rowsAffectedCount != 1 {
		logger.Error("Did not insert new user.", zap.Int64("rows_affected", rowsAffectedCount))
		return "", "", false, status.Error(codes.Internal, "Error finding or creating user account.")
	}

	return userID, username, true, nil
}

func AuthenticateXPID(ctx context.Context, logger *zap.Logger, db *sql.DB, xpID evr.EvrId) (string, string, error) {
	userID, username, _, err := AuthenticateDevice(ctx, logger, db, DeviceXPID(xpID), "", false)
	return userID, username, err
}

func AuthenticateVRMLUser(ctx context.Context, logger *zap.Logger, db *sql.DB, vrmlUserID string) (string, string, error) {
	userID, username, _, err := AuthenticateDevice(ctx, logger, db, DeviceVRMLUserID(vrmlUserID), "", false)
	return userID, username, err
}

// verifyJWT parses and verifies a JWT token using the provided key function.
// It returns the parsed token if it is valid, otherwise it returns an error.
// Nakama JWT's are signed by the `session.session_encryption_key` in the Nakama config.
func VerifySignedJWT(rawToken string, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

// GetPlayerByXPID retrieves the EVR profile associated with a given XPID.
func GetPlayerByXPID(ctx context.Context, db *sql.DB, nk runtime.NakamaModule, xpid evr.EvrId) (*EVRProfile, error) {
	found := true
	// Look for an existing account.
	query := "SELECT user_id FROM user_device WHERE id = $1"
	var dbUserID string
	err := db.QueryRowContext(ctx, query, DeviceXPID(xpid)).Scan(&dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			found = false
		} else {
			return nil, status.Error(codes.Internal, "Error finding user account by device id.")
		}
	}
	if found {
		if account, err := nk.AccountGetId(ctx, dbUserID); err != nil {
			return nil, status.Error(codes.Internal, "Error finding user account by device id.")
		} else {
			return BuildEVRProfileFromAccount(account)
		}
	}
	return nil, status.Error(codes.NotFound, "User account not found.")
}

// GetUserIDByXPID retrieves the user ID associated with a given XPID.
func GetUserIDByXPID(ctx context.Context, db *sql.DB, xpid evr.EvrId) (string, error) {
	query := `
	SELECT ud.user_id FROM user_device ud WHERE ud.id = $1`
	var dbUserID string
	var found = true
	err := db.QueryRowContext(ctx, query, DeviceXPID(xpid)).Scan(&dbUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			found = false
		} else {
			return "", fmt.Errorf("error finding user ID By Evr ID: %w", err)
		}
	}
	if !found {
		return "", status.Error(codes.NotFound, "user account not found")
	}
	if dbUserID == "" {
		return "", nil
	}
	return dbUserID, nil
}
