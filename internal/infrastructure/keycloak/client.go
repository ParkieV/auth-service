package keycloak

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ParkieV/auth-service/internal/config"
)

type Client struct {
	baseURL      string
	realm        string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewClient(cfg config.KeycloakConfig) *Client {
	return &Client{
		baseURL:      cfg.URL,
		realm:        cfg.Realm,
		clientID:     cfg.ClientID,
		clientSecret: cfg.Secret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Authenticate(username, password string) (string, string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)
	form := url.Values{
		"grant_type":    {"password"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"username":      {username},
		"password":      {password},
	}
	resp, err := c.httpClient.PostForm(endpoint, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("authentication failed: %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", "", err
	}
	return tok.AccessToken, tok.RefreshToken, nil
}

func (c *Client) RefreshToken(refreshToken string) (string, string, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"refresh_token": {refreshToken},
	}
	resp, err := c.httpClient.PostForm(endpoint, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("refresh failed: %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", "", err
	}
	return tok.AccessToken, tok.RefreshToken, nil
}
