package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/heroiclabs/nakama/v3/server/evr"
)

// linkHeadset links a headset to a user account.
func (d *DiscordAppBot) linkHeadset(ctx context.Context, db *sql.DB, logger runtime.Logger, discordID, linkCode string) (err error) {
	nk := d.nk

	// Exchange the code for the data
	xpID, clientIP, payload, err := ExchangeLinkCode(ctx, nk, logger, linkCode)
	if err != nil {
		return fmt.Errorf("failed to exchange link code: %w", err)
	}

	// Authenticate/create an account.
	userID, _, _, err := AuthenticateDiscord(ctx, RuntimeLoggerToZapLogger(logger), db, discordID, true)
	if err != nil {
		return fmt.Errorf("failed to authenticate (or create) user %s: %w", discordID, err)
	}

	// Link the headset to the user account.
	if err := LinkXPID(ctx, RuntimeLoggerToZapLogger(logger), db, uuid.FromStringOrNil(userID), xpID); err != nil {
		return fmt.Errorf("failed to link headset: %w", err)
	}

	// Validate the XPID
	profile := evr.LoginProfile{}
	if err := json.Unmarshal([]byte(payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal login profile: %w", err)
	}

	// Set the client IP as authorized in the LoginHistory
	history := &LoginHistory{}
	if err := StorageRead(ctx, nk, userID, history, true); err != nil {
		return fmt.Errorf("failed to load login history: %w", err)
	}
	history.Update(xpID, clientIP, &profile, true)

	// Save the login history.
	if err := StorageWrite(ctx, nk, userID, history); err != nil {
		return fmt.Errorf("failed to save login history: %w", err)
	}

	// Record the link event in metrics.
	tags := map[string]string{
		"headset_type": normalizeHeadsetType(profile.SystemInfo.HeadsetType),
		"is_pcvr":      fmt.Sprintf("%t", profile.BuildNumber != evr.StandaloneBuildNumber),
		"new_account":  "false",
	}

	if gg, ok := ctx.Value(ctxGuildGroupKey{}).(*GuildGroup); ok && gg != nil {
		// Add the user to the group.
		if err := d.nk.GroupUsersAdd(ctx, SystemUserID, gg.IDStr(), []string{userID}); err != nil {
			return fmt.Errorf("error joining group: %w", err)
		}
		tags["guild_id"] = gg.GuildID
	}

	d.metrics.CustomCounter("link_headset", tags, 1)
	return nil
}

// handleLinkHeadsetInteraction handles the interaction for linking a headset. (both command and modal)
func (d *DiscordAppBot) handleLinkHeadsetInteraction(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, code string) error {
	code = strings.TrimSpace(code)
	code = strings.ToUpper(code)

	// Validate the link code as a 4 character string
	var reason string
	var embed *discordgo.MessageEmbed
	if len(code) != 4 {
		embed = &discordgo.MessageEmbed{
			Title:       "Invalid Code",
			Description: "The code must be exactly 4 characters long. (e.g. `ABCD`). Please try again.",
			Color:       0xCC0000, // Red
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Detailed Error",
					Value:  fmt.Sprintf("```fix\n%s\n```", reason),
					Inline: false,
				},
			},
		}
	} else {
		if err := d.linkHeadset(ctx, d.db, logger, i.Member.User.ID, code); err != nil {
			if err == ErrLinkNotFound {
				embed = &discordgo.MessageEmbed{
					Title:       "Linking Failed",
					Description: "The link code is invalid or has expired. Please try again.",
					Color:       0xCC0000, // Red
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Detailed Error",
							Value:  fmt.Sprintf("```fix\n%s\n```", reason),
							Inline: false,
						},
					},
				}
			} else {
				embed = &discordgo.MessageEmbed{
					Title:       "Linking Failed",
					Description: "An error occurred while linking your headset. Please try again later.",
					Color:       0xCC0000, // Red
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Detailed Error",
							Value:  fmt.Sprintf("```fix\n%s\n```", err.Error()),
							Inline: false,
						},
					},
				}
			}
		} else {
			embed = &discordgo.MessageEmbed{
				Title:       "Headset Linked",
				Description: "Your headset has been linked. Restart your game.",
				Color:       0x00CC66, // Green
			}
		}
	}

	d.cache.QueueSyncMember(i.GuildID, i.Member.User.ID, false)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				embed,
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

}

func (d *DiscordAppBot) handleUnlinkHeadset(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, user *discordgo.User, member *discordgo.Member, userID string, groupID string) error {
	nk := d.nk
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {

		account, err := nk.AccountGetId(ctx, userID)
		if err != nil {
			logger.WithField("error", err).Error("Failed to get account")
			return err
		}
		if len(account.Devices) == 0 {
			return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   discordgo.MessageFlagsEphemeral,
					Content: "No headsets are linked to this account.",
				},
			})
		}

		loginHistory := NewLoginHistory(userID)
		if err := StorageRead(ctx, nk, userID, loginHistory, true); err != nil {
			logger.WithField("error", err).Error("Failed to load login history")
			return err
		}
		xpids := make([]string, 0, len(account.Devices))
		for _, d := range account.Devices {
			if xpidStr, ok := strings.CutPrefix(d.Id, DevicePrefixXPID); ok {
				xpids = append(xpids, xpidStr)
			}
		}

		options := make([]discordgo.SelectMenuOption, 0, len(xpids))
		for _, xpid := range xpids {

			description := ""

			v, _ := evr.ParseEvrId(xpid)
			if v == nil {
				continue
			}
			if ts, ok := loginHistory.GetXPI(*v); ok {
				hours := int(time.Since(ts).Hours())
				if hours < 1 {
					minutes := int(time.Since(ts).Minutes())
					if minutes < 1 {
						description = "Just now"
					} else {
						description = fmt.Sprintf("%d minutes ago", minutes)
					}
				} else if hours < 24 {
					description = fmt.Sprintf("%d hours ago", hours)
				} else {
					description = fmt.Sprintf("%d days ago", int(time.Since(ts).Hours()/24))
				}
			}

			options = append(options, discordgo.SelectMenuOption{
				Label: xpid,
				Value: xpid,
				Emoji: &discordgo.ComponentEmoji{
					Name: "🔗",
				},
				Description: description,
			})
		}

		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: "Select a device to unlink",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								// Select menu, as other components, must have a customID, so we set it to this value.
								CustomID:    "unlink-headset",
								Placeholder: "<select a device to unlink>",
								Options:     options,
							},
						},
					},
				},
			},
		}
		return s.InteractionRespond(i.Interaction, response)
	}
	xpid := options[0].StringValue()
	// Validate the link code as a 4 character string

	if user == nil {
		return nil
	}

	if err := func() error {
		xpid, err := evr.ParseEvrId(xpid)
		if err != nil {
			return fmt.Errorf("failed to parse XPID: %w", err)
		}
		return UnlinkXPID(ctx, RuntimeLoggerToZapLogger(logger), d.db, uuid.FromStringOrNil(userID), *xpid)

	}(); err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: err.Error(),
			},
		})
	}
	d.metrics.CustomCounter("unlink_headset", nil, 1)
	content := "Your headset has been unlinked. Restart your game."
	d.cache.QueueSyncMember(i.GuildID, user.ID, false)

	if err := d.cache.updateLinkStatus(ctx, user.ID); err != nil {
		return fmt.Errorf("failed to update link status: %w", err)
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: content,
		},
	})
}
