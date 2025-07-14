package server

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/heroiclabs/nakama-common/runtime"
)

func (d *DiscordAppBot) handleSuspend(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, user *discordgo.User, member *discordgo.Member, userID string, groupID string) error {

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Suspend Player",
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							MenuType:    discordgo.UserSelectMenu,
							CustomID:    "suspend_player_select",
							Placeholder: "Select a player",
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Lookup",
							Style:    discordgo.SecondaryButton,
							CustomID: "suspend_player_lookup",
							Disabled: true,
						},
						discordgo.Button{
							Label:    "Select Player",
							Style:    discordgo.PrimaryButton,
							CustomID: "suspend_player_confirm",
						},
					},
				},
			},
		},
	})
}

func (d *DiscordAppBot) handleSuspendPlayerSelect(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	targetUser, err := s.User(targetUserID)
	if err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Suspend Player",
					Author: &discordgo.MessageEmbedAuthor{
						Name:    targetUser.Username,
						IconURL: targetUser.AvatarURL(""),
					},
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Lookup",
							Style:    discordgo.SecondaryButton,
							CustomID: "suspend_player_lookup:" + targetUserID,
						},
						discordgo.Button{
							Label:    "Select Player",
							Style:    discordgo.PrimaryButton,
							CustomID: "suspend_player_confirm:" + targetUserID,
						},
					},
				},
			},
		},
	})
}

func (d *DiscordAppBot) handleSuspendPlayerLookup(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	target, err := s.User(targetUserID)
	if err != nil {
		return err
	}

	opts := UserProfileRequestOptions{
		IncludeSuspensionsEmbed:       true,
		IncludePastSuspensions:        true,
		IncludeInactiveSuspensions:    true,
		IncludeSuspensionAuditorNotes: true,
	}

	return d.handleProfileRequest(ctx, logger, d.nk, s, i, target, opts)
}

func (d *DiscordAppBot) handleSuspendPlayerConfirm(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	targetUser, err := s.User(targetUserID)
	if err != nil {
		return err
	}

	profile, err := EVRProfileLoad(ctx, d.nk, d.cache.DiscordIDToUserID(targetUserID))
	if err != nil {
		return err
	}

	guildGroup, ok := ctx.Value(ctxGuildGroupKey{}).(*GuildGroup)
	if !ok {
		return errors.New("failed to retrieve guild group from context")
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Suspend Player",
					Author: &discordgo.MessageEmbedAuthor{
						Name:    targetUser.Username,
						IconURL: targetUser.AvatarURL(""),
					},
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "IGN",
							Value: profile.GetGroupDisplayNameOrDefault(guildGroup.IDStr()),
						},
					},
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Temp Kick/Ban",
							Style:    discordgo.DangerButton,
							CustomID: "suspend_player_temp_ban:" + targetUserID,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "suspend_duration",
							Label:     "Suspension Duration",
							Style:     discordgo.TextInputShort,
							Value:     "1h",
							Required:  true,
							MaxLength: 10,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "suspend_message",
							Label:     "User Message",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MaxLength: 42,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Add Moderator Notes...",
							Style:    discordgo.SecondaryButton,
							CustomID: "suspend_player_add_notes:" + targetUserID,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID: "suspend_scope",
							Options: []discordgo.SelectMenuOption{
								{
									Label: "Public Only",
									Value: "public",
								},
								{
									Label: "All Lobby/Matches",
									Value: "all",
								},
							},
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Activate Suspension",
							Style:    discordgo.SuccessButton,
							CustomID: "suspend_player_activate:" + targetUserID,
						},
						discordgo.Button{
							Label:    "Cancel",
							Style:    discordgo.SecondaryButton,
							CustomID: "suspend_player_cancel",
						},
					},
				},
			},
		},
	})
}

func (d *DiscordAppBot) handleSuspendPlayerTempBan(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	targetUser, err := s.User(targetUserID)
	if err != nil {
		return err
	}
	member, ok := ctx.Value(ctxMemberKey{}).(*discordgo.Member)
	if !ok {
		return errors.New("failed to retrieve member from context")
	}
	return d.kickPlayer(logger, i, member, targetUser, "5m", "Temporary suspension.", "Temporary suspension triggered from /suspend command.", false, false)
}

func (d *DiscordAppBot) handleSuspendPlayerAddNotes(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "suspend_player_notes_modal:" + targetUserID,
			Title:    "Moderator Notes",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "notes",
							Label:     "Notes",
							Style:     discordgo.TextInputParagraph,
							Required:  true,
							MaxLength: 1024,
						},
					},
				},
			},
		},
	})
}

func (d *DiscordAppBot) handleSuspendPlayerActivate(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, targetUserID string) error {
	data := i.MessageComponentData()

	duration := data.Values[0]
	message := data.Values[1]
	notes := ""
	if len(data.Values) > 2 {
		notes = data.Values[2]
	}
	scope := data.Values[3]

	allowPrivateLobbyAccess := scope == "public"

	targetUser, err := s.User(targetUserID)
	if err != nil {
		return err
	}
	member := ctx.Value(ctxMemberKey{}).(*discordgo.Member)

	err = d.kickPlayer(logger, i, member, targetUser, duration, message, notes, false, allowPrivateLobbyAccess)
	if err != nil {
		return err
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Suspension Activated",
					Description: "The suspension has been activated.",
				},
			},
		},
	})
}
