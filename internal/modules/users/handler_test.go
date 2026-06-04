package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockUserService struct {
	preRegisterUserFunc  func(ctx context.Context, input PreRegisterInput) error
	verifyEmailFunc      func(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error)
	completeRegisterFunc func(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error)
	loginUserFunc        func(ctx context.Context, input LoginInput) (LoginResult, error)
	refreshTokenFunc     func(ctx context.Context, input RefreshInput) (RefreshResult, error)
	logoutUserFunc       func(ctx context.Context, input RefreshInput) error
	getProfileFunc       func(ctx context.Context, uuidStr string) (AuthUser, error)
	updateProfileFunc    func(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error)
	deleteUserFunc       func(ctx context.Context, uuidStr string) error
}

func (m *mockUserService) PreRegisterUser(ctx context.Context, input PreRegisterInput) error {
	if m.preRegisterUserFunc != nil {
		return m.preRegisterUserFunc(ctx, input)
	}
	return nil
}

func (m *mockUserService) VerifyEmail(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error) {
	if m.verifyEmailFunc != nil {
		return m.verifyEmailFunc(ctx, input)
	}
	return VerifyEmailResult{}, nil
}

func (m *mockUserService) CompleteRegister(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error) {
	if m.completeRegisterFunc != nil {
		return m.completeRegisterFunc(ctx, input)
	}
	return RegisterResult{}, nil
}

func (m *mockUserService) LoginUser(ctx context.Context, input LoginInput) (LoginResult, error) {
	if m.loginUserFunc != nil {
		return m.loginUserFunc(ctx, input)
	}
	return LoginResult{}, nil
}

func (m *mockUserService) RefreshToken(ctx context.Context, input RefreshInput) (RefreshResult, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, input)
	}
	return RefreshResult{}, nil
}

func (m *mockUserService) LogoutUser(ctx context.Context, input RefreshInput) error {
	if m.logoutUserFunc != nil {
		return m.logoutUserFunc(ctx, input)
	}
	return nil
}

func (m *mockUserService) GetProfile(ctx context.Context, uuidStr string) (AuthUser, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, uuidStr)
	}
	return AuthUser{}, nil
}

func (m *mockUserService) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, uuidStr, input)
	}
	return UpdateProfileResult{}, nil
}

func (m *mockUserService) DeleteUser(ctx context.Context, uuidStr string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, uuidStr)
	}
	return nil
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestHandler_PreRegister(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{name: "happy path", body: `{"email":"Ada@Example.COM"}`, wantStatus: http.StatusOK},
		{name: "invalid body", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "email taken", body: `{"email":"ada@example.com"}`, serviceErr: ErrEmailTaken, wantStatus: http.StatusConflict},
		{name: "internal error", body: `{"email":"ada@example.com"}`, serviceErr: errors.New("send failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockUserService{
				preRegisterUserFunc: func(ctx context.Context, input PreRegisterInput) error {
					if tt.wantStatus != http.StatusBadRequest && input.Email != "ada@example.com" {
						t.Fatalf("email = %q, want normalized ada@example.com", input.Email)
					}
					return tt.serviceErr
				},
			}

			rec := callJSONHandler(http.MethodPost, "/users/pre-register", tt.body, NewHandler(service, testCookieConfig()).PreRegister)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_VerifyEmail(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantToken  bool
	}{
		{name: "happy path", body: `{"email":"Ada@Example.COM","code":"123456"}`, wantStatus: http.StatusOK, wantToken: true},
		{name: "invalid body", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "expired code", body: `{"email":"ada@example.com","code":"123456"}`, serviceErr: ErrCodeExpired, wantStatus: http.StatusGone},
		{name: "invalid code", body: `{"email":"ada@example.com","code":"000000"}`, serviceErr: ErrCodeInvalid, wantStatus: http.StatusUnauthorized},
		{name: "not found", body: `{"email":"ada@example.com","code":"123456"}`, serviceErr: ErrVerificationNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockUserService{
				verifyEmailFunc: func(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error) {
					if tt.wantStatus != http.StatusBadRequest && input.Email != "ada@example.com" {
						t.Fatalf("email = %q, want normalized ada@example.com", input.Email)
					}
					return VerifyEmailResult{VerificationToken: "token"}, tt.serviceErr
				},
			}

			rec := callJSONHandler(http.MethodPost, "/users/verify-email", tt.body, NewHandler(service, testCookieConfig()).VerifyEmail)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantToken {
				var result VerifyEmailResult
				if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if result.VerificationToken != "token" {
					t.Fatalf("verification token = %q", result.VerificationToken)
				}
			}
		})
	}
}

func TestHandler_RegisterCompletesRegistration(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "happy path",
			body:       `{"verification_token":"token","first_name":" Ada ","last_name":" Lovelace ","password":"supersecret","phone":" 123 "}`,
			wantStatus: http.StatusCreated,
		},
		{name: "invalid body", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "short password", body: `{"verification_token":"token","first_name":"Ada","last_name":"Lovelace","password":"short"}`, wantStatus: http.StatusBadRequest},
		{name: "invalid token", body: `{"verification_token":"token","first_name":"Ada","last_name":"Lovelace","password":"supersecret"}`, serviceErr: ErrInvalidVerificationToken, wantStatus: http.StatusUnauthorized},
		{name: "email taken", body: `{"verification_token":"token","first_name":"Ada","last_name":"Lovelace","password":"supersecret"}`, serviceErr: ErrEmailTaken, wantStatus: http.StatusConflict},
		{name: "internal error", body: `{"verification_token":"token","first_name":"Ada","last_name":"Lovelace","password":"supersecret"}`, serviceErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockUserService{
				completeRegisterFunc: func(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error) {
					if tt.wantStatus != http.StatusBadRequest {
						if input.FirstName != "Ada" || input.LastName != "Lovelace" {
							t.Fatalf("names were not trimmed: %+v", input)
						}
					}
					return RegisterResult{Message: registerSuccessMessage, User: AuthUser{UserID: 1, Email: "ada@example.com"}}, tt.serviceErr
				},
				loginUserFunc: func(ctx context.Context, input LoginInput) (LoginResult, error) {
					if input.Email != "ada@example.com" || input.Password != "supersecret" {
						t.Fatalf("login input = %+v", input)
					}
					return LoginResult{AccessToken: "access-token", RefreshToken: "refresh-token"}, nil
				},
			}

			rec := callJSONHandler(http.MethodPost, "/users/register", tt.body, NewHandler(service, testCookieConfig()).Register)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusCreated {
				assertCookie(t, rec, "spazio_access_token", "access-token")
				assertCookie(t, rec, "spazio_refresh_token", "refresh-token")
			}
		})
	}
}

func TestHandler_RefreshUsesRefreshCookie(t *testing.T) {
	service := &mockUserService{
		refreshTokenFunc: func(ctx context.Context, input RefreshInput) (RefreshResult, error) {
			if input.RefreshToken != "old-refresh-token" {
				t.Fatalf("refresh token = %q", input.RefreshToken)
			}
			return RefreshResult{AccessToken: "new-access-token", RefreshToken: "new-refresh-token"}, nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/refresh", "", NewHandler(service, testCookieConfig()).Refresh, withCookie("spazio_refresh_token", "old-refresh-token"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	assertCookie(t, rec, "spazio_access_token", "new-access-token")
	assertCookie(t, rec, "spazio_refresh_token", "new-refresh-token")
}

func TestHandler_LogoutUsesRefreshCookieAndClearsCookies(t *testing.T) {
	service := &mockUserService{
		logoutUserFunc: func(ctx context.Context, input RefreshInput) error {
			if input.RefreshToken != "refresh-token" {
				t.Fatalf("refresh token = %q", input.RefreshToken)
			}
			return nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/logout", "", NewHandler(service, testCookieConfig()).Logout, withCookie("spazio_refresh_token", "refresh-token"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	assertCookieMaxAge(t, rec, "spazio_access_token", -1)
	assertCookieMaxAge(t, rec, "spazio_refresh_token", -1)
}

func TestHandler_GetProfile(t *testing.T) {
	tests := []struct {
		name       string
		userUUID   string
		serviceErr error
		wantStatus int
	}{
		{name: "happy path", userUUID: "8a6fbb17-b64b-4f40-a09d-b6639b357ef5", wantStatus: http.StatusOK},
		{name: "missing auth context", wantStatus: http.StatusUnauthorized},
		{name: "user not found", userUUID: "8a6fbb17-b64b-4f40-a09d-b6639b357ef5", serviceErr: ErrUserNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockUserService{
				getProfileFunc: func(ctx context.Context, uuidStr string) (AuthUser, error) {
					if uuidStr != tt.userUUID {
						t.Fatalf("uuidStr = %q, want %q", uuidStr, tt.userUUID)
					}
					return AuthUser{UserID: 1, UserUUID: uuidStr, Email: "ada@example.com", RoleID: 1, RoleName: "Admin"}, tt.serviceErr
				},
			}

			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/users/profile", nil)
			if tt.userUUID != "" {
				c.Set("user_uuid", tt.userUUID)
			}

			NewHandler(service, testCookieConfig()).GetProfile(c)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func callJSONHandler(method, path, body string, handler gin.HandlerFunc, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		opt(req)
	}
	c.Request = req
	handler(c)
	return rec
}

func testCookieConfig() CookieConfig {
	return CookieConfig{
		AccessTokenMaxAge:  3600,
		RefreshTokenMaxAge: refreshTokenMaxAgeSeconds,
	}
}

func withCookie(name, value string) func(*http.Request) {
	return func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
}

func assertCookie(t *testing.T, rec *httptest.ResponseRecorder, name, value string) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			if cookie.Value != value {
				t.Fatalf("%s value = %q, want %q", name, cookie.Value, value)
			}
			if !cookie.HttpOnly {
				t.Fatalf("%s should be httpOnly", name)
			}
			return
		}
	}
	t.Fatalf("cookie %s was not set", name)
}

func assertCookieMaxAge(t *testing.T, rec *httptest.ResponseRecorder, name string, maxAge int) {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			if cookie.MaxAge != maxAge {
				t.Fatalf("%s maxAge = %d, want %d", name, cookie.MaxAge, maxAge)
			}
			return
		}
	}
	t.Fatalf("cookie %s was not set", name)
}
