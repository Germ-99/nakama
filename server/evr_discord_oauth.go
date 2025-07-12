package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/heroiclabs/nakama-common/runtime"
	"golang.org/x/oauth2"
)

const (
	AuthenticationTokenExpirySeconds = 15 * 60 // 15 minutes

)

// DiscordAccessToken represents the Discord access token structure.
type DiscordAccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (t *DiscordAccessToken) Config() *oauth2.Config {
	return &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discord.com/api/oauth2/authorize",
			TokenURL:  "https://discord.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func (t *DiscordAccessToken) Refresh(clientId string, clientSecret string) error {
	oauthUrl := "https://discord.com/api/v10/oauth2/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", t.RefreshToken)
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", oauthUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord refresh failed: %s", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return err
	}

	return nil
}

func ExchangeCodeForAccessToken(logger runtime.Logger, code string, clientId string, clientSecret string, redirectUrl string) (*DiscordAccessToken, error) {

	conf := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discord.com/api/oauth2/authorize",
			TokenURL:  "https://discord.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes:       []string{"identify"},
		RedirectURL:  redirectUrl,
		ClientID:     clientId,
		ClientSecret: clientSecret,
	}

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}
	accessToken := &DiscordAccessToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.Expiry.Second(),
	}

	return accessToken, nil
}

type DiscordSignInRpcRequest struct {
	Code             string `json:"code"`
	OAuthRedirectUrl string `json:"oauth_redirect_url"`
}

type DiscordSignInRpcResponse struct {
	SessionToken    string `json:"sessionToken"`
	DiscordUsername string `json:"discordUsername"`
}

// DiscordSignInRpc handles the Discord sign-in RPC call.
func DiscordSignInRpc(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	vars, _ := ctx.Value(runtime.RUNTIME_CTX_ENV).(map[string]string)
	clientId := vars["DISCORD_CLIENT_ID"]
	clientSecret := vars["DISCORD_CLIENT_SECRET"]
	userID := ""

	// Parse the payload into a LoginRequest object
	var request DiscordSignInRpcRequest
	if err := json.Unmarshal([]byte(payload), &request); err != nil {
		logger.WithField("err", err).WithField("payload", payload).Error("Unable to unmarshal payload")
		return "", runtime.NewError("Unable to unmarshal payload", StatusInvalidArgument)
	}
	if request.Code == "" {
		logger.Error("DiscordSignInRpc: Code is empty")
		return "", runtime.NewError("Code is empty", StatusInvalidArgument)
	}
	if request.OAuthRedirectUrl == "" {
		logger.Error("DiscordSignInRpc: OAuthRedirectUrl is empty")
		return "", runtime.NewError("OAuthRedirectUrl is empty", StatusInvalidArgument)
	}

	// Exchange the code for an access token
	accessToken, err := ExchangeCodeForAccessToken(logger, request.Code, clientId, clientSecret, request.OAuthRedirectUrl)
	if err != nil {
		logger.WithField("err", err).Error("Unable to exchange code for access token")
		return "", runtime.NewError("Unable to exchange code for access token", StatusInternalError)
	}

	// Create a Discord client
	discord, err := discordgo.New("Bearer " + accessToken.AccessToken)
	if err != nil {
		logger.WithField("err", err).Error("Unable to create Discord client")
		return "", runtime.NewError("Unable to create Discord client", StatusInternalError)
	}

	// Get the Discord user
	user, err := discord.User("@me")
	if err != nil {
		logger.WithField("err", err).Error("Unable to get Discord user")
		return "", runtime.NewError("Unable to get Discord user", StatusInternalError)
	}

	// Authenticate/create an account.
	userID, username, _, err := AuthenticateDiscord(ctx, nk.(*RuntimeGoNakamaModule).logger, db, user.ID, true)
	if err != nil {
		logger.WithField("err", err).Error("Unable to authenticate or create user")
		return "", runtime.NewError("Unable to authenticate or create user", StatusInternalError)
	}

	expiry := time.Now().UTC().Unix() + 15*60 // 15 minutes
	// Generate a session token for the user to use to authenticate for device linking
	sessionToken, _, err := nk.AuthenticateTokenGenerate(userID, username, expiry, nil)
	if err != nil {
		logger.WithField("err", err).Error("Unable to generate session token")
		return "", runtime.NewError("Unable to generate session token", StatusInternalError)
	}

	// store the jwt in the user's metadata so it can be can verified later
	response := DiscordSignInRpcResponse{
		SessionToken:    sessionToken,
		DiscordUsername: user.Username,
	}

	responsejson, err := json.Marshal(response)
	if err != nil {
		return "", runtime.NewError(fmt.Sprintf("error marshalling LoginSuccess response: %v", err), StatusInternalError)
	}

	return string(responsejson), nil
}
