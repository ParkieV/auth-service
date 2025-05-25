package auth_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ParkieV/auth-service/internal/config"
)

type AuthClient interface {
	GenerateTokens(ctx context.Context, userID string) (string, string, error)
	IssueAccessToken(ctx context.Context, userID string) (string, error)
	VerifyAccess(ctx context.Context, accessToken string) (bool, string, error)
	Logout(ctx context.Context, refreshToken string) error
}

type KeycloakClient struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	log          *slog.Logger
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type introspectResp struct {
	Active bool   `json:"active"`
	Sub    string `json:"sub"`
}

func NewKeycloakClient(cfg config.KeycloakConfig, log *slog.Logger) *KeycloakClient {
	return &KeycloakClient{
		baseURL:      cfg.URL,
		realm:        cfg.Realm,
		clientID:     cfg.ClientID,
		clientSecret: cfg.Secret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		log:          log,
	}
}

func (kc *KeycloakClient) GenerateTokens(ctx context.Context, userID string) (string, string, error) {
	return kc.tokenExchange(ctx, userID, true)
}

func (kc *KeycloakClient) IssueAccessToken(ctx context.Context, userID string) (string, error) {
	access, _, err := kc.tokenExchange(ctx, userID, false)
	return access, err
}

func (kc *KeycloakClient) VerifyAccess(ctx context.Context, access string) (bool, string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token/introspect",
		kc.baseURL, kc.realm)

	form := url.Values{
		"token":         {access},
		"client_id":     {kc.clientID},
		"client_secret": {kc.clientSecret},
	}

	var out introspectResp
	if err := kc.postFormJSON(ctx, endpoint, form, &out); err != nil {
		return false, "", err
	}
	return out.Active, out.Sub, nil
}

func (kc *KeycloakClient) Logout(ctx context.Context, refresh string) error {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout",
		kc.baseURL, kc.realm)

	form := url.Values{
		"client_id":     {kc.clientID},
		"client_secret": {kc.clientSecret},
		"refresh_token": {refresh},
	}

	resp, err := kc.postForm(ctx, endpoint, form)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed: %d", resp.StatusCode)
	}
	return nil
}

func (kc *KeycloakClient) tokenExchange(ctx context.Context, userID string, needRefresh bool) (string, string, error) {
	adminAT, err := kc.clientCredentials(ctx)
	if err != nil {
		return "", "", err
	}

	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		kc.baseURL, kc.realm)

	form := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"client_id":          {kc.clientID},
		"client_secret":      {kc.clientSecret},
		"subject_token":      {adminAT},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"requested_subject":  {userID},
	}
	if needRefresh {
		form.Set("requested_token_type", "urn:ietf:params:oauth:token-type:refresh_token")
	}

	var out tokenResp
	if err := kc.postFormJSON(ctx, endpoint, form, &out); err != nil {
		return "", "", err
	}
	if out.AccessToken == "" {
		return "", "", errors.New("no access token in response")
	}
	return out.AccessToken, out.RefreshToken, nil
}

func (kc *KeycloakClient) clientCredentials(ctx context.Context) (string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		kc.baseURL, kc.realm)

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {kc.clientID},
		"client_secret": {kc.clientSecret},
	}

	var out tokenResp
	if err := kc.postFormJSON(ctx, endpoint, form, &out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", errors.New("no access token in client_credentials response")
	}
	return out.AccessToken, nil
}

func (kc *KeycloakClient) postFormJSON(ctx context.Context, endpoint string, form url.Values, dst any) error {
	resp, err := kc.postForm(ctx, endpoint, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("keycloak: %s (%d): %s", endpoint, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (kc *KeycloakClient) postForm(ctx context.Context, endpoint string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return kc.httpClient.Do(req)
}
