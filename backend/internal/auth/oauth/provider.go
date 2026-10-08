// Package oauth implements the authorization-code flow (with PKCE) for the
// "Sign in with Google / Microsoft" buttons. It only talks to the provider;
// account lookup and session creation live in the auth service.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Firakef1/settle/backend/internal/shared/config"
)

// Profile is what we need from the provider to sign someone in.
type Profile struct {
	Subject       string // stable provider user ID
	Email         string
	EmailVerified bool
	Name          string
}

// Provider holds one provider's endpoints and credentials.
type Provider struct {
	Name         string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
	ExtraParams  map[string]string
	HTTPClient   *http.Client
	parseProfile func(map[string]any) Profile
}

var ErrNoEmail = errors.New("the provider did not return an email address")

// AuthCodeURL is where the browser is sent to sign in at the provider.
func (p *Provider) AuthCodeURL(state, redirectURI, codeChallenge string) string {
	q := url.Values{
		"client_id":             {p.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(p.Scopes, " ")},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	for k, v := range p.ExtraParams {
		q.Set(k, v)
	}
	return p.AuthURL + "?" + q.Encode()
}

// Exchange trades the authorization code for an access token.
func (p *Provider) Exchange(ctx context.Context, code, redirectURI, codeVerifier string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"code_verifier": {codeVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := p.doJSON(req, &body); err != nil {
		return "", fmt.Errorf("%s token exchange: %w", p.Name, err)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("%s token exchange: %s %s", p.Name, body.Error, body.Description)
	}
	return body.AccessToken, nil
}

// FetchProfile reads the signed-in user's identity from the userinfo endpoint.
func (p *Provider) FetchProfile(ctx context.Context, accessToken string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return Profile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	claims := map[string]any{}
	if err := p.doJSON(req, &claims); err != nil {
		return Profile{}, fmt.Errorf("%s userinfo: %w", p.Name, err)
	}
	profile := p.parseProfile(claims)
	profile.Email = strings.ToLower(strings.TrimSpace(profile.Email))
	if profile.Subject == "" {
		return Profile{}, fmt.Errorf("%s userinfo: missing subject", p.Name)
	}
	if profile.Email == "" {
		return Profile{}, ErrNoEmail
	}
	return profile, nil
}

func (p *Provider) doJSON(req *http.Request, out any) error {
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, out)
}

func str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// truthy accepts both true and "true" (some providers send booleans as strings).
func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true")
	}
	return false
}

// NewGoogle returns the Google provider (OpenID Connect).
func NewGoogle(clientID, clientSecret string) *Provider {
	return &Provider{
		Name:         "google",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		ExtraParams:  map[string]string{"prompt": "select_account"},
		parseProfile: func(c map[string]any) Profile {
			return Profile{Subject: str(c, "sub"), Email: str(c, "email"), EmailVerified: truthy(c["email_verified"]), Name: str(c, "name")}
		},
	}
}

// NewMicrosoft returns the Microsoft identity platform provider. Microsoft
// doesn't say whether the email is verified, so it is treated as unverified:
// it can create a new account but won't be linked to an existing one.
func NewMicrosoft(clientID, clientSecret, tenant string) *Provider {
	if tenant == "" {
		tenant = "common"
	}
	base := "https://login.microsoftonline.com/" + url.PathEscape(tenant) + "/oauth2/v2.0"
	return &Provider{
		Name:         "microsoft",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      base + "/authorize",
		TokenURL:     base + "/token",
		UserInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		ExtraParams:  map[string]string{"prompt": "select_account"},
		parseProfile: func(c map[string]any) Profile {
			name := str(c, "name")
			if name == "" {
				name = strings.TrimSpace(str(c, "given_name") + " " + str(c, "family_name"))
			}
			return Profile{Subject: str(c, "sub"), Email: str(c, "email"), EmailVerified: false, Name: name}
		},
	}
}

// FromConfig returns the providers that have credentials configured.
func FromConfig(cfg config.OAuthConfig) map[string]*Provider {
	providers := map[string]*Provider{}
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providers["google"] = NewGoogle(cfg.GoogleClientID, cfg.GoogleClientSecret)
	}
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		providers["microsoft"] = NewMicrosoft(cfg.MicrosoftClientID, cfg.MicrosoftClientSecret, cfg.MicrosoftTenant)
	}
	return providers
}

// NewPKCE returns a code verifier and its S256 challenge.
func NewPKCE() (verifier, challenge string, err error) {
	verifier, err = RandomToken(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// RandomToken returns n random bytes, base64url-encoded.
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
