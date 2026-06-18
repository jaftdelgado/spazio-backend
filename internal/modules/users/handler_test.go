package users

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type mockUserService struct {
	preRegisterUserFunc         func(ctx context.Context, input PreRegisterInput) error
	verifyEmailFunc             func(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error)
	completeRegisterFunc        func(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error)
	loginUserFunc               func(ctx context.Context, input LoginInput) (LoginResult, error)
	refreshTokenFunc            func(ctx context.Context, input RefreshInput) (RefreshResult, error)
	logoutUserFunc              func(ctx context.Context, input RefreshInput) error
	requestPasswordResetFunc    func(ctx context.Context, input ForgotPasswordInput) error
	verifyPasswordResetCodeFunc func(ctx context.Context, input VerifyPasswordResetCodeInput) (PasswordResetVerificationResult, error)
	resetPasswordFunc           func(ctx context.Context, input ResetPasswordInput) error
	getProfileFunc              func(ctx context.Context, uuidStr string) (UserProfile, error)
	listAgentsFunc              func(ctx context.Context) (ListAgentsResult, error)
	updateProfileFunc           func(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error)
	uploadProfilePhotoFunc      func(ctx context.Context, input UploadProfilePhotoInput) (UpdateProfileResult, error)
	requestEmailChangeFunc      func(ctx context.Context, uuidStr string, input RequestEmailChangeInput) error
	verifyEmailChangeFunc       func(ctx context.Context, uuidStr string, input VerifyEmailChangeInput) (EmailChangeVerificationResult, error)
	confirmEmailChangeFunc      func(ctx context.Context, uuidStr string, input ConfirmEmailChangeInput) (UpdateProfileResult, error)
	changePasswordFunc          func(ctx context.Context, uuidStr string, input ChangePasswordInput) error
	adminCreateUserFunc         func(ctx context.Context, input AdminCreateUserInput) (AdminCreateUserResult, error)
	deleteUserFunc              func(ctx context.Context, uuidStr string) error
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

func (m *mockUserService) RequestPasswordReset(ctx context.Context, input ForgotPasswordInput) error {
	if m.requestPasswordResetFunc != nil {
		return m.requestPasswordResetFunc(ctx, input)
	}
	return nil
}

func (m *mockUserService) VerifyPasswordResetCode(ctx context.Context, input VerifyPasswordResetCodeInput) (PasswordResetVerificationResult, error) {
	if m.verifyPasswordResetCodeFunc != nil {
		return m.verifyPasswordResetCodeFunc(ctx, input)
	}
	return PasswordResetVerificationResult{}, nil
}

func (m *mockUserService) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	if m.resetPasswordFunc != nil {
		return m.resetPasswordFunc(ctx, input)
	}
	return nil
}

func (m *mockUserService) GetProfile(ctx context.Context, uuidStr string) (UserProfile, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, uuidStr)
	}
	return UserProfile{}, nil
}

func (m *mockUserService) ListAgents(ctx context.Context) (ListAgentsResult, error) {
	if m.listAgentsFunc != nil {
		return m.listAgentsFunc(ctx)
	}
	return ListAgentsResult{}, nil
}

func (m *mockUserService) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, uuidStr, input)
	}
	return UpdateProfileResult{}, nil
}

func (m *mockUserService) UploadProfilePhoto(ctx context.Context, input UploadProfilePhotoInput) (UpdateProfileResult, error) {
	if m.uploadProfilePhotoFunc != nil {
		return m.uploadProfilePhotoFunc(ctx, input)
	}
	return UpdateProfileResult{}, nil
}

func (m *mockUserService) RequestEmailChange(ctx context.Context, uuidStr string, input RequestEmailChangeInput) error {
	if m.requestEmailChangeFunc != nil {
		return m.requestEmailChangeFunc(ctx, uuidStr, input)
	}
	return nil
}

func (m *mockUserService) VerifyEmailChange(ctx context.Context, uuidStr string, input VerifyEmailChangeInput) (EmailChangeVerificationResult, error) {
	if m.verifyEmailChangeFunc != nil {
		return m.verifyEmailChangeFunc(ctx, uuidStr, input)
	}
	return EmailChangeVerificationResult{}, nil
}

func (m *mockUserService) ConfirmEmailChange(ctx context.Context, uuidStr string, input ConfirmEmailChangeInput) (UpdateProfileResult, error) {
	if m.confirmEmailChangeFunc != nil {
		return m.confirmEmailChangeFunc(ctx, uuidStr, input)
	}
	return UpdateProfileResult{}, nil
}

func (m *mockUserService) ChangePassword(ctx context.Context, uuidStr string, input ChangePasswordInput) error {
	if m.changePasswordFunc != nil {
		return m.changePasswordFunc(ctx, uuidStr, input)
	}
	return nil
}

func (m *mockUserService) AdminCreateUser(ctx context.Context, input AdminCreateUserInput) (AdminCreateUserResult, error) {
	if m.adminCreateUserFunc != nil {
		return m.adminCreateUserFunc(ctx, input)
	}
	return AdminCreateUserResult{}, nil
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
	service := &mockUserService{
		preRegisterUserFunc: func(ctx context.Context, input PreRegisterInput) error {
			if input.Email != "ada@example.com" {
				t.Fatalf("email = %q", input.Email)
			}
			return nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/pre-register", `{"email":"Ada@Example.COM"}`, NewHandler(service, testCookieConfig()).PreRegister)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_RegisterCompletesRegistration(t *testing.T) {
	service := &mockUserService{
		completeRegisterFunc: func(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error) {
			if input.FirstName != "Ada" || input.LastName != "Lovelace" {
				t.Fatalf("names were not trimmed: %+v", input)
			}
			return RegisterResult{Message: registerSuccessMessage, User: AuthUser{UserID: 1, Email: "ada@example.com"}}, nil
		},
		loginUserFunc: func(ctx context.Context, input LoginInput) (LoginResult, error) {
			return LoginResult{AccessToken: "access-token", RefreshToken: "refresh-token"}, nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/register", `{"verification_token":"token","first_name":" Ada ","last_name":" Lovelace ","password":"supersecret"}`, NewHandler(service, testCookieConfig()).Register)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	assertCookie(t, rec, "spazio_access_token", "access-token")
	assertCookie(t, rec, "spazio_refresh_token", "refresh-token")
}

func TestHandler_RequestPasswordReset(t *testing.T) {
	service := &mockUserService{
		requestPasswordResetFunc: func(ctx context.Context, input ForgotPasswordInput) error {
			if input.Email != "ada@example.com" {
				t.Fatalf("email = %q", input.Email)
			}
			return nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/forgot-password", `{"email":"Ada@Example.com"}`, NewHandler(service, testCookieConfig()).RequestPasswordReset)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_VerifyPasswordResetCode(t *testing.T) {
	service := &mockUserService{
		verifyPasswordResetCodeFunc: func(ctx context.Context, input VerifyPasswordResetCodeInput) (PasswordResetVerificationResult, error) {
			if input.Email != "ada@example.com" || input.Code != "123456" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return PasswordResetVerificationResult{ResetToken: "reset-token"}, nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/forgot-password/verify", `{"email":"Ada@Example.com","code":"123456"}`, NewHandler(service, testCookieConfig()).VerifyPasswordResetCode)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var result PasswordResetVerificationResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil || result.ResetToken != "reset-token" {
		t.Fatalf("unexpected result: %+v err=%v", result, err)
	}
}

func TestHandler_ResetPassword(t *testing.T) {
	var called bool
	service := &mockUserService{
		resetPasswordFunc: func(ctx context.Context, input ResetPasswordInput) error {
			called = true
			if input.ResetToken != "reset-token" || input.NewPassword != "new-secret-123" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/forgot-password/reset", `{"reset_token":"reset-token","new_password":"new-secret-123"}`, NewHandler(service, testCookieConfig()).ResetPassword)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
}

func TestHandler_GetProfile(t *testing.T) {
	service := &mockUserService{
		getProfileFunc: func(ctx context.Context, uuidStr string) (UserProfile, error) {
			return UserProfile{UserID: 1, UserUUID: uuidStr, Email: "ada@example.com", FirstName: "Ada", LastName: "Lovelace"}, nil
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	c.Set("user_uuid", "user-uuid")

	NewHandler(service, testCookieConfig()).GetProfile(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UpdateProfile(t *testing.T) {
	service := &mockUserService{
		updateProfileFunc: func(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error) {
			if input.FirstName != "Ada" || input.LastName != "Lovelace" || input.Phone != "123" {
				t.Fatalf("unexpected input: %+v", input)
			}
			return UpdateProfileResult{Message: profileUpdatedMessage, User: UserProfile{UserUUID: uuidStr}}, nil
		},
	}

	rec := callAuthedJSONHandler(http.MethodPut, "/users/profile", `{"first_name":" Ada ","last_name":" Lovelace ","phone":" 123 "}`, "user-uuid", NewHandler(service, testCookieConfig()).UpdateProfile)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_ListAgents(t *testing.T) {
	service := &mockUserService{
		listAgentsFunc: func(ctx context.Context) (ListAgentsResult, error) {
			return ListAgentsResult{
				Data: []AgentListItem{{
					UserID:    21,
					UserUUID:  "agent-uuid",
					FirstName: "Ada",
					LastName:  "Lovelace",
				}},
			}, nil
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/agents", nil)

	NewHandler(service, testCookieConfig()).ListAgents(c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}

	var result ListAgentsResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].UserID != 21 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHandler_UploadProfilePhoto(t *testing.T) {
	var called bool
	service := &mockUserService{
		uploadProfilePhotoFunc: func(ctx context.Context, input UploadProfilePhotoInput) (UpdateProfileResult, error) {
			called = true
			if input.UserUUID != "user-uuid" || input.File == nil {
				t.Fatalf("unexpected upload input: %+v", input)
			}
			return UpdateProfileResult{Message: profileUpdatedMessage, User: UserProfile{UserUUID: input.UserUUID}}, nil
		},
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("CreateFormFile(): %v", err)
	}
	if _, err := part.Write([]byte("fake-image")); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/users/profile/photo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req
	c.Set("user_uuid", "user-uuid")

	NewHandler(service, testCookieConfig()).UploadProfilePhoto(c)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
}

func TestHandler_RequestEmailChange(t *testing.T) {
	var called bool
	service := &mockUserService{
		requestEmailChangeFunc: func(ctx context.Context, uuidStr string, input RequestEmailChangeInput) error {
			called = true
			if uuidStr != "user-uuid" || input.NewEmail != "new@example.com" {
				t.Fatalf("unexpected input: %s %+v", uuidStr, input)
			}
			return nil
		},
	}

	rec := callAuthedJSONHandler(http.MethodPost, "/users/profile/email/request", `{"new_email":"new@example.com"}`, "user-uuid", NewHandler(service, testCookieConfig()).RequestEmailChange)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
}

func TestHandler_ConfirmEmailChange(t *testing.T) {
	service := &mockUserService{
		confirmEmailChangeFunc: func(ctx context.Context, uuidStr string, input ConfirmEmailChangeInput) (UpdateProfileResult, error) {
			if uuidStr != "user-uuid" || input.VerificationToken != "change-token" {
				t.Fatalf("unexpected input: %s %+v", uuidStr, input)
			}
			return UpdateProfileResult{Message: emailChangedMessage, User: UserProfile{UserUUID: uuidStr, Email: "new@example.com"}}, nil
		},
	}

	rec := callAuthedJSONHandler(http.MethodPut, "/users/profile/email", `{"verification_token":"change-token"}`, "user-uuid", NewHandler(service, testCookieConfig()).ConfirmEmailChange)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_ChangePassword(t *testing.T) {
	var called bool
	service := &mockUserService{
		changePasswordFunc: func(ctx context.Context, uuidStr string, input ChangePasswordInput) error {
			called = true
			if uuidStr != "user-uuid" || input.CurrentPassword != "current" || input.NewPassword != "new-secret-123" {
				t.Fatalf("unexpected input: %s %+v", uuidStr, input)
			}
			return nil
		},
	}

	rec := callAuthedJSONHandler(http.MethodPut, "/users/profile/password", `{"current_password":"current","new_password":"new-secret-123"}`, "user-uuid", NewHandler(service, testCookieConfig()).ChangePassword)
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
}

func TestHandler_AdminCreateUser(t *testing.T) {
	service := &mockUserService{
		adminCreateUserFunc: func(ctx context.Context, input AdminCreateUserInput) (AdminCreateUserResult, error) {
			if input.FirstName != "Grace" || input.LastName != "Hopper" || input.Email != "grace@example.com" || input.RoleID != roleIDAgent {
				t.Fatalf("unexpected input: %+v", input)
			}
			return AdminCreateUserResult{Message: adminUserCreatedMessage, TemporaryPassword: "Temp1234", User: AuthUser{UserID: 1}}, nil
		},
	}

	rec := callJSONHandler(http.MethodPost, "/users/staff", `{"first_name":" Grace ","last_name":" Hopper ","email":"grace@example.com","role_id":2}`, NewHandler(service, testCookieConfig()).AdminCreateUser)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
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

func TestHandler_WriteUserError_CurrentPasswordInvalid(t *testing.T) {
	service := &mockUserService{
		changePasswordFunc: func(ctx context.Context, uuidStr string, input ChangePasswordInput) error {
			return ErrCurrentPasswordInvalid
		},
	}

	rec := callAuthedJSONHandler(http.MethodPut, "/users/profile/password", `{"current_password":"bad","new_password":"new-secret-123"}`, "user-uuid", NewHandler(service, testCookieConfig()).ChangePassword)
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "contraseña actual") {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
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

func callAuthedJSONHandler(method, path, body, userUUID string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if userUUID != "" {
		c.Set("user_uuid", userUUID)
	}
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
