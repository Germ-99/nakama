package server

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/heroiclabs/nakama-common/runtime"
)

func (d *DiscordAppBot) handleGameServiceCommand(ctx context.Context, logger runtime.Logger, s *discordgo.Session, i *discordgo.InteractionCreate, command string) error {

	var (
		userID   = ctx.Value(ctxUserIDKey{}).(string)
		username = ctx.Value(ctxUsernameKey{}).(string)
		data     = i.Interaction.MessageComponentData()
	)

	switch command {
	case "command-select":
		// Handle the command select interaction
		selectedOption := data.Values[0]

		switch selectedOption {
		case "generate-client-token":
			// Never expires
			encKey := d.nk.(*RuntimeGoNakamaModule).config.GetSession().EncryptionKey
			token, tokenID, err := generateClientToken(encKey, userID, username)
			if err != nil {
				return fmt.Errorf("failed to generate auth token: %w", err)
			}

			if err := LinkClientTokenID(ctx, d.zapLogger, d.db, uuid.FromStringOrNil(userID), tokenID); err != nil {
				return fmt.Errorf("failed to link client token: %w", err)
			}

			// Create an embed with the new token
			embed := &discordgo.MessageEmbed{
				Title: "Game Client Authentication Token",
				Description: "Here is your game client auth token for use in `config.json`. This token does not expire. Please keep it safe." +
					fmt.Sprintf("```fix\n%s\n```", token),
			}

			// Add a new response
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:  discordgo.MessageFlagsEphemeral,
					Embeds: []*discordgo.MessageEmbed{embed},
				},
			}); err != nil {
				return fmt.Errorf("failed to send embed message: %w", err)
			}
			return nil
		}
	}
	return nil
}

type gameServiceCommand struct{}

var GameServiceCommand = &gameServiceCommand{}
