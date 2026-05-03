package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthConfig *oauth2.Config

const oauthStateKey = "oauth_state"

func InitOAuth() {
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

type GoogleUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"picture"`
}

func GetAuthURL(w http.ResponseWriter, r *http.Request) string {
	state := generateState()
	sess, _ := store.Get(r, sessionName)
	sess.Values[oauthStateKey] = state
	sess.Save(r, w)
	return oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func ExchangeCode(ctx context.Context, w http.ResponseWriter, r *http.Request, code, state string) (*GoogleUser, error) {
	sess, err := store.Get(r, sessionName)
	if err != nil {
		return nil, fmt.Errorf("session error")
	}
	savedState, _ := sess.Values[oauthStateKey].(string)
	if savedState != state {
		return nil, fmt.Errorf("invalid oauth state")
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	client := oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("get userinfo: %w", err)
	}
	defer resp.Body.Close()

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &user, nil
}

func generateState() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = "abcdefghijklmnopqrstuvwxyz0123456789"[i%36]
	}
	return string(b)
}
