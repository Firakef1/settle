package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Firakef1/settle/backend/internal/auth/oauth"
	"github.com/Firakef1/settle/backend/internal/auth/service"
)

const (
	oauthCookieName = "settle_oauth"
	oauthCookiePath = "/api/v1/auth/oauth"
	oauthCookieTTL  = 10 * time.Minute
)

// OAuthHandler serves the "Sign in with Google / Microsoft" redirects.
type OAuthHandler struct {
	svc           *service.OAuthService
	publicBaseURL string // where the browser reaches the API (redirect URIs)
	frontendURL   string // where users land afterwards
	secret        []byte // signs the state cookie
}

// NewOAuthHandler creates an OAuthHandler.
func NewOAuthHandler(svc *service.OAuthService, publicBaseURL, frontendURL, secret string) *OAuthHandler {
	return &OAuthHandler{
		svc:           svc,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
		frontendURL:   strings.TrimRight(frontendURL, "/"),
		secret:        []byte(secret),
	}
}

// oauthState travels in a signed, HttpOnly cookie between start and callback.
type oauthState struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Provider string `json:"p"`
	Next     string `json:"n"`
	Expires  int64  `json:"e"`
}

func (h *OAuthHandler) redirectURI(provider string) string {
	return h.publicBaseURL + "/api/v1/auth/oauth/" + provider + "/callback"
}

// safeNext only allows same-site paths ("/dashboard", not "//evil.com").
func safeNext(next string) string {
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") && !strings.HasPrefix(next, "/\\") {
		return next
	}
	return ""
}

func (h *OAuthHandler) sign(payload []byte) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *OAuthHandler) verify(value string) (*oauthState, bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(payload)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(got, mac.Sum(nil)) {
		return nil, false
	}
	var st oauthState
	if json.Unmarshal(payload, &st) != nil || time.Now().Unix() > st.Expires {
		return nil, false
	}
	return &st, true
}

func (h *OAuthHandler) setCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthCookieName,
		Value:    value,
		Path:     oauthCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   strings.HasPrefix(h.publicBaseURL, "https://"),
		// Lax: the cookie must come back on the provider's top-level redirect.
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *OAuthHandler) fail(c *gin.Context, code string) {
	c.Redirect(http.StatusFound, h.frontendURL+"/login?oauth_error="+url.QueryEscape(code))
}

// Providers godoc
// @Summary      List enabled sign-in providers
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string][]string
// @Router       /auth/oauth/providers [get]
func (h *OAuthHandler) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": h.svc.Enabled()})
}

// Start godoc
// @Summary      Start signing in with a provider
// @Description  Redirects the browser to Google or Microsoft. Optional `next` is a path to open afterwards.
// @Tags         auth
// @Param        provider  path   string  true   "google or microsoft"
// @Param        next      query  string  false  "Path to open after sign-in"
// @Success      302
// @Router       /auth/oauth/{provider}/start [get]
func (h *OAuthHandler) Start(c *gin.Context) {
	name := c.Param("provider")
	provider, err := h.svc.Provider(name)
	if err != nil {
		h.fail(c, "unavailable")
		return
	}
	state, err := oauth.RandomToken(24)
	if err != nil {
		h.fail(c, "failed")
		return
	}
	verifier, challenge, err := oauth.NewPKCE()
	if err != nil {
		h.fail(c, "failed")
		return
	}
	payload, _ := json.Marshal(oauthState{
		State:    state,
		Verifier: verifier,
		Provider: name,
		Next:     safeNext(c.Query("next")),
		Expires:  time.Now().Add(oauthCookieTTL).Unix(),
	})
	h.setCookie(c, h.sign(payload), int(oauthCookieTTL.Seconds()))
	c.Redirect(http.StatusFound, provider.AuthCodeURL(state, h.redirectURI(name), challenge))
}

// Callback godoc
// @Summary      Provider redirect target
// @Description  Exchanges the code, signs the user in and redirects to the frontend with the session in the URL fragment.
// @Tags         auth
// @Param        provider  path   string  true  "google or microsoft"
// @Success      302
// @Router       /auth/oauth/{provider}/callback [get]
func (h *OAuthHandler) Callback(c *gin.Context) {
	name := c.Param("provider")
	cookie, _ := c.Cookie(oauthCookieName)
	h.setCookie(c, "", -1) // single use

	if c.Query("error") != "" {
		// e.g. the user pressed Cancel on the consent screen
		h.fail(c, "cancelled")
		return
	}
	st, ok := h.verify(cookie)
	if !ok || st.Provider != name || st.State == "" || !hmac.Equal([]byte(st.State), []byte(c.Query("state"))) {
		h.fail(c, "state")
		return
	}
	provider, err := h.svc.Provider(name)
	if err != nil {
		h.fail(c, "unavailable")
		return
	}

	ctx := c.Request.Context()
	accessToken, err := provider.Exchange(ctx, c.Query("code"), h.redirectURI(name), st.Verifier)
	if err != nil {
		log.Printf("[ERROR] oauth %s exchange: %v", name, err)
		h.fail(c, "failed")
		return
	}
	profile, err := provider.FetchProfile(ctx, accessToken)
	if err != nil {
		log.Printf("[ERROR] oauth %s profile: %v", name, err)
		if errors.Is(err, oauth.ErrNoEmail) {
			h.fail(c, "no_email")
			return
		}
		h.fail(c, "failed")
		return
	}
	session, err := h.svc.SignIn(ctx, name, profile)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOAuthAccountExists):
			h.fail(c, "account_exists")
		default:
			log.Printf("[ERROR] oauth %s sign-in: %v", name, err)
			h.fail(c, "failed")
		}
		return
	}

	// Tokens go in the fragment: browsers never send it to a server or log it.
	fragment := url.Values{"token": {session.Token}, "refresh_token": {session.RefreshToken}}
	if st.Next != "" {
		fragment.Set("next", st.Next)
	}
	c.Redirect(http.StatusFound, h.frontendURL+"/oauth/callback#"+fragment.Encode())
}
