package keycloak

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ParkieV/auth-service/internal/config"
)

// Client взаимодействует с Keycloak OIDC endpoints
type Client struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewClient создаёт обёртку над HTTP-клиентом
func NewClient(cfg config.KeycloakConfig) *Client {
	return &Client{
		baseURL:      cfg.URL,
		realm:        cfg.Realm,
		clientID:     cfg.ClientID,
		clientSecret: cfg.Secret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Authenticate выполняет grant_type=password
func (c *Client) Authenticate(username, password string) (string, string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("username", username)
	form.Set("password", password)

	resp, err := c.httpClient.PostForm(endpoint, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.AccessToken, out.RefreshToken, nil
}

// RefreshToken выполняет grant_type=refresh_token
func (c *Client) RefreshToken(refreshToken string) (string, string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("refresh_token", refreshToken)

	resp, err := c.httpClient.PostForm(endpoint, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("refresh failed: status %d", resp.StatusCode)
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.AccessToken, out.RefreshToken, nil
}
