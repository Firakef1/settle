package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/model"
	"github.com/Firakef1/settle/backend/internal/auth/oauth"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

// fakeProvider is a minimal OAuth server: /token checks PKCE, /userinfo returns `profile`.
type fakeProvider struct {
	server    *httptest.Server
	challenge string // last code_challenge seen by the test
	profile   map[string]any
}

func newFakeProvider(t *testing.T) *fakeProvider {
	fp := &fakeProvider{profile: map[string]any{"sub": "g-123", "email": "grace@lab.test", "email_verified": true, "name": "Grace Hopper"}}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if r.Form.Get("code") != "good-code" || base64.RawURLEncoding.EncodeToString(sum[:]) != fp.challenge {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"provider-access-token","token_type":"Bearer"}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer provider-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(fp.profile)
	})
	fp.server = httptest.NewServer(mux)
	t.Cleanup(fp.server.Close)
	return fp
}

type oauthTestEnv struct {
	router   *gin.Engine
	users    *repository.UserRepo
	provider *fakeProvider
}

func setupOAuthTest(t *testing.T) *oauthTestEnv {
	fp := newFakeProvider(t)
	google := oauth.NewGoogle("client-id", "client-secret")
	google.AuthURL = fp.server.URL + "/authorize"
	google.TokenURL = fp.server.URL + "/token"
	google.UserInfoURL = fp.server.URL + "/userinfo"

	users := repository.NewUserRepo(nil)
	hashSvc := sharedService.NewHashService()
	authSvc := service.NewAuthService(users, repository.NewRefreshTokenRepo(nil), repository.NewPasswordResetOTPRepo(nil),
		&noopEmailSender{}, hashSvc, sharedService.NewJWTServiceWithSecret("oauth_test_secret"))
	oauthSvc := service.NewOAuthService(map[string]*oauth.Provider{"google": google}, users, repository.NewIdentityRepo(nil), authSvc, hashSvc)
	h := NewOAuthHandler(oauthSvc, "http://app.test", "http://app.test", "cookie-signing-secret")

	r := gin.New()
	r.GET("/api/v1/auth/oauth/providers", h.Providers)
	r.GET("/api/v1/auth/oauth/:provider/start", h.Start)
	r.GET("/api/v1/auth/oauth/:provider/callback", h.Callback)
	return &oauthTestEnv{router: r, users: users, provider: fp}
}

func (e *oauthTestEnv) get(path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// start runs /start and returns the state cookie and the state sent to the provider.
func (e *oauthTestEnv) start(t *testing.T, query string) (*http.Cookie, string) {
	w := e.get("/api/v1/auth/oauth/google/start" + query)
	require.Equal(t, http.StatusFound, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "http://app.test/api/v1/auth/oauth/google/callback", loc.Query().Get("redirect_uri"))
	assert.Equal(t, "S256", loc.Query().Get("code_challenge_method"))
	e.provider.challenge = loc.Query().Get("code_challenge")

	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == oauthCookieName {
			cookie = c
		}
	}
	require.NotNil(t, cookie, "state cookie should be set")
	assert.True(t, cookie.HttpOnly)
	return cookie, loc.Query().Get("state")
}

func fragment(t *testing.T, location string) url.Values {
	t.Helper()
	require.True(t, strings.HasPrefix(location, "http://app.test/oauth/callback#"), location)
	v, err := url.ParseQuery(strings.SplitN(location, "#", 2)[1])
	require.NoError(t, err)
	return v
}

func TestOAuthProviders(t *testing.T) {
	e := setupOAuthTest(t)
	w := e.get("/api/v1/auth/oauth/providers")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":["google"]}`, w.Body.String())
}

func TestOAuthStartUnknownProvider(t *testing.T) {
	e := setupOAuthTest(t)
	w := e.get("/api/v1/auth/oauth/github/start")
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "http://app.test/login?oauth_error=unavailable", w.Header().Get("Location"))
}

func TestOAuthNewUserFullFlow(t *testing.T) {
	e := setupOAuthTest(t)
	cookie, state := e.start(t, "?next=/requests/new")

	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, cookie)
	require.Equal(t, http.StatusFound, w.Code)
	f := fragment(t, w.Header().Get("Location"))
	assert.NotEmpty(t, f.Get("token"))
	assert.NotEmpty(t, f.Get("refresh_token"))
	assert.Equal(t, "/requests/new", f.Get("next"))

	user, err := e.users.FindByEmail(context.Background(), "grace@lab.test")
	require.NoError(t, err)
	assert.Equal(t, "Grace Hopper", user.Name)
	assert.True(t, user.EmailVerified, "provider-verified email should be verified")

	// Signing in again with the same provider account reuses the user.
	cookie2, state2 := e.start(t, "")
	w2 := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state2, cookie2)
	require.Equal(t, http.StatusFound, w2.Code)
	fragment(t, w2.Header().Get("Location"))
	again, err := e.users.FindByEmail(context.Background(), "grace@lab.test")
	require.NoError(t, err)
	assert.Equal(t, user.ID, again.ID)
}

func TestOAuthLinksExistingAccountWhenEmailVerified(t *testing.T) {
	e := setupOAuthTest(t)
	existing := &model.User{ID: "u-1", Email: "grace@lab.test", Name: "Grace", PasswordHash: "x", Status: "active", EmailVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, e.users.CreateUser(context.Background(), existing))

	cookie, state := e.start(t, "")
	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, cookie)
	require.Equal(t, http.StatusFound, w.Code)
	fragment(t, w.Header().Get("Location"))
}

func TestOAuthRefusesToLinkUnverifiedProviderEmail(t *testing.T) {
	e := setupOAuthTest(t)
	e.provider.profile["email_verified"] = false
	existing := &model.User{ID: "u-1", Email: "grace@lab.test", Name: "Grace", PasswordHash: "x", Status: "active", EmailVerified: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, e.users.CreateUser(context.Background(), existing))

	cookie, state := e.start(t, "")
	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, cookie)
	assert.Equal(t, "http://app.test/login?oauth_error=account_exists", w.Header().Get("Location"))
}

func TestOAuthRejectsBadState(t *testing.T) {
	e := setupOAuthTest(t)
	cookie, _ := e.start(t, "")

	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state=forged", cookie)
	assert.Equal(t, "http://app.test/login?oauth_error=state", w.Header().Get("Location"))

	// No cookie at all.
	w = e.get("/api/v1/auth/oauth/google/callback?code=good-code&state=anything")
	assert.Equal(t, "http://app.test/login?oauth_error=state", w.Header().Get("Location"))

	// Tampered cookie.
	tampered := *cookie
	tampered.Value = cookie.Value[:len(cookie.Value)-2] + "xx"
	_, state := e.start(t, "")
	w = e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, &tampered)
	assert.Equal(t, "http://app.test/login?oauth_error=state", w.Header().Get("Location"))
}

func TestOAuthBadCodeFails(t *testing.T) {
	e := setupOAuthTest(t)
	cookie, state := e.start(t, "")
	w := e.get("/api/v1/auth/oauth/google/callback?code=wrong&state="+state, cookie)
	assert.Equal(t, "http://app.test/login?oauth_error=failed", w.Header().Get("Location"))
}

func TestOAuthCancelled(t *testing.T) {
	e := setupOAuthTest(t)
	cookie, state := e.start(t, "")
	w := e.get("/api/v1/auth/oauth/google/callback?error=access_denied&state="+state, cookie)
	assert.Equal(t, "http://app.test/login?oauth_error=cancelled", w.Header().Get("Location"))
}

func TestOAuthIgnoresUnsafeNext(t *testing.T) {
	e := setupOAuthTest(t)
	cookie, state := e.start(t, "?next=//evil.example/steal")
	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, cookie)
	f := fragment(t, w.Header().Get("Location"))
	assert.Empty(t, f.Get("next"))
}

func TestOAuthVerifiesSquattedAccountAndReplacesPassword(t *testing.T) {
	e := setupOAuthTest(t)
	hash, err := sharedService.NewHashService().Hash("squatter-password")
	require.NoError(t, err)
	squatted := &model.User{ID: "u-sq", Email: "grace@lab.test", Name: "Not Grace", PasswordHash: hash, Status: "active", EmailVerified: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, e.users.CreateUser(context.Background(), squatted))

	cookie, state := e.start(t, "")
	w := e.get("/api/v1/auth/oauth/google/callback?code=good-code&state="+state, cookie)
	fragment(t, w.Header().Get("Location"))

	user, err := e.users.FindByEmail(context.Background(), "grace@lab.test")
	require.NoError(t, err)
	assert.True(t, user.EmailVerified)
	assert.False(t, sharedService.NewHashService().Compare("squatter-password", user.PasswordHash), "the squatter's password must stop working")
}
