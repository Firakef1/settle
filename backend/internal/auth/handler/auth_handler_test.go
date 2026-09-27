package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/auth/dto"
	"github.com/Firakef1/settle/backend/internal/auth/repository"
	"github.com/Firakef1/settle/backend/internal/auth/service"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupHandlerTest() (*AuthHandler, *repository.UserRepo, *repository.RefreshTokenRepo) {
	userRepo := repository.NewUserRepo(nil)
	refreshTokenRepo := repository.NewRefreshTokenRepo(nil)
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTServiceWithSecret("handler_test_secret")
	svc := service.NewAuthService(userRepo, refreshTokenRepo, hashSvc, jwtSvc)
	handler := NewAuthHandler(svc)
	return handler, userRepo, refreshTokenRepo
}

func performRequest(handler gin.HandlerFunc, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.Handle(method, path, handler)

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_Signup_Success(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	w := performRequest(handler.Signup, "POST", "/signup", req, nil)

	assert.Equal(t, http.StatusCreated, w.Code)

	var res map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Equal(t, "user created successfully", res["message"])

	userData := res["user"].(map[string]interface{})
	assert.Equal(t, "test@example.com", userData["email"])
	assert.Equal(t, "Test User", userData["name"])
	assert.NotEmpty(t, userData["id"])
}

func TestHandler_Signup_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Signup, "POST", "/signup", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Signup_ValidationError(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Password: "password123",
		Name:     "Test User",
	}

	w := performRequest(handler.Signup, "POST", "/signup", req, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Signup_DuplicateEmail(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	req := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	performRequest(handler.Signup, "POST", "/signup", req, nil)

	w := performRequest(handler.Signup, "POST", "/signup", req, nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Login_Success(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	w := performRequest(handler.Login, "POST", "/login", loginReq, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var res dto.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.NotEmpty(t, res.RefreshToken)
	assert.Equal(t, "test@example.com", res.User.Email)
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Login, "POST", "/login", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Login_InvalidCredentials(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	w := performRequest(handler.Login, "POST", "/login", loginReq, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Refresh_Success(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	refreshReq := dto.RefreshRequest{
		RefreshToken: loginRes.RefreshToken,
	}

	w := performRequest(handler.Refresh, "POST", "/refresh", refreshReq, nil)

	assert.Equal(t, http.StatusOK, w.Code)

	var res dto.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestHandler_Refresh_InvalidJSON(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	w := performRequest(handler.Refresh, "POST", "/refresh", "invalid json", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Refresh_InvalidToken(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	refreshReq := dto.RefreshRequest{
		RefreshToken: "bogustoken",
	}

	w := performRequest(handler.Refresh, "POST", "/refresh", refreshReq, nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Logout_Success(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	logoutReq := dto.LogoutRequest{
		RefreshToken: loginRes.RefreshToken,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + loginRes.Token,
	}

	w := performRequest(handler.Logout, "POST", "/logout", logoutReq, headers)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Logout_NoBody(t *testing.T) {
	handler, _, _ := setupHandlerTest()

	signupReq := dto.SignupRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}
	performRequest(handler.Signup, "POST", "/signup", signupReq, nil)

	loginReq := dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	wLogin := performRequest(handler.Login, "POST", "/login", loginReq, nil)
	var loginRes dto.LoginResponse
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)

	headers := map[string]string{
		"Authorization": "Bearer " + loginRes.Token,
	}

	w := performRequest(handler.Logout, "POST", "/logout", nil, headers)

	assert.Equal(t, http.StatusOK, w.Code)
}
