package users

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	createUserFunc                      func(ctx context.Context, input CreateUserRecord) (AuthUser, error)
	getUserByEmailFunc                  func(ctx context.Context, email string) (UserAuthRecord, error)
	getUserByUUIDFunc                   func(ctx context.Context, uuidStr string) (UserAuthRecord, error)
	getUserByIDFunc                     func(ctx context.Context, userID int32) (UserAuthRecord, error)
	updateUserStatusFunc                func(ctx context.Context, userID int32, statusID int32) error
	updateProfileFunc                   func(ctx context.Context, uuidStr string, input UpdateProfileInput) (AuthUser, error)
	softDeleteUserFunc                  func(ctx context.Context, uuidStr string) error
	createPendingVerificationFunc       func(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error)
	getLatestPendingVerificationFunc    func(ctx context.Context, email string) (PendingVerification, error)
	markPendingVerificationVerifiedFunc func(ctx context.Context, verificationID int32) error
	createRefreshTokenFunc              func(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error
	getRefreshTokenFunc                 func(ctx context.Context, tokenHash string) (RefreshTokenRecord, error)
	revokeRefreshTokenFunc              func(ctx context.Context, tokenHash string) error
	revokeAllUserRefreshTokensFunc      func(ctx context.Context, userID int32) error
}

func (m *mockUserRepository) CreateUser(ctx context.Context, input CreateUserRecord) (AuthUser, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, input)
	}
	return AuthUser{}, nil
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (UserAuthRecord, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return UserAuthRecord{}, ErrUserNotFound
}

func (m *mockUserRepository) GetUserByUUID(ctx context.Context, uuidStr string) (UserAuthRecord, error) {
	if m.getUserByUUIDFunc != nil {
		return m.getUserByUUIDFunc(ctx, uuidStr)
	}
	return UserAuthRecord{}, nil
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, userID int32) (UserAuthRecord, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, userID)
	}
	return UserAuthRecord{}, nil
}

func (m *mockUserRepository) UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error {
	if m.updateUserStatusFunc != nil {
		return m.updateUserStatusFunc(ctx, userID, statusID)
	}
	return nil
}

func (m *mockUserRepository) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (AuthUser, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, uuidStr, input)
	}
	return AuthUser{}, nil
}

func (m *mockUserRepository) SoftDeleteUser(ctx context.Context, uuidStr string) error {
	if m.softDeleteUserFunc != nil {
		return m.softDeleteUserFunc(ctx, uuidStr)
	}
	return nil
}

func (m *mockUserRepository) CreatePendingVerification(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error) {
	if m.createPendingVerificationFunc != nil {
		return m.createPendingVerificationFunc(ctx, email, codeHash, expiresAt)
	}
	return 1, nil
}

func (m *mockUserRepository) GetLatestPendingVerification(ctx context.Context, email string) (PendingVerification, error) {
	if m.getLatestPendingVerificationFunc != nil {
		return m.getLatestPendingVerificationFunc(ctx, email)
	}
	return PendingVerification{}, nil
}

func (m *mockUserRepository) MarkPendingVerificationVerified(ctx context.Context, verificationID int32) error {
	if m.markPendingVerificationVerifiedFunc != nil {
		return m.markPendingVerificationVerifiedFunc(ctx, verificationID)
	}
	return nil
}

func (m *mockUserRepository) CreateRefreshToken(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error {
	if m.createRefreshTokenFunc != nil {
		return m.createRefreshTokenFunc(ctx, userID, tokenHash, expiresAt)
	}
	return nil
}

func (m *mockUserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	if m.getRefreshTokenFunc != nil {
		return m.getRefreshTokenFunc(ctx, tokenHash)
	}
	return RefreshTokenRecord{}, nil
}

func (m *mockUserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if m.revokeRefreshTokenFunc != nil {
		return m.revokeRefreshTokenFunc(ctx, tokenHash)
	}
	return nil
}

func (m *mockUserRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID int32) error {
	if m.revokeAllUserRefreshTokensFunc != nil {
		return m.revokeAllUserRefreshTokensFunc(ctx, userID)
	}
	return nil
}

type mockEmailSender struct {
	code string
	to   string
	err  error
}

func (m *mockEmailSender) SendVerificationCode(to, code, appName string) error {
	m.to = to
	m.code = code
	return m.err
}

type mockJWTService struct {
	token string
	err   error
}

func (m mockJWTService) Generate(userID int32, userUUID string, roleID int32, roleName string) (string, error) {
	return m.token, m.err
}

func (m mockJWTService) Validate(tokenString string) (*auth.Claims, error) {
	return nil, errors.New("not used")
}

func newTestService(repo UserRepository, emailSender *mockEmailSender) *service {
	svc := NewService(repo, emailSender, mockJWTService{token: "access-token"}, "Spazio", "test-secret").(*service)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	}
	return svc
}

func TestService_PreRegisterUser(t *testing.T) {
	tests := []struct {
		name      string
		userErr   error
		emailErr  error
		wantErr   error
		wantEmail bool
	}{
		{name: "happy path sends code", userErr: ErrUserNotFound, wantEmail: true},
		{name: "email already registered", userErr: nil, wantErr: ErrEmailTaken},
		{name: "email sender failure logs code and continues", userErr: ErrUserNotFound, emailErr: errors.New("resend down"), wantEmail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emailSender := &mockEmailSender{err: tt.emailErr}
			var storedEmail string
			var storedCodeHash string
			var storedExpiresAt time.Time
			repo := &mockUserRepository{
				getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
					if email != "ada@example.com" {
						t.Fatalf("email = %q, want normalized ada@example.com", email)
					}
					if tt.userErr == nil {
						return UserAuthRecord{Email: email}, nil
					}
					return UserAuthRecord{}, tt.userErr
				},
				createPendingVerificationFunc: func(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error) {
					storedEmail = email
					storedCodeHash = codeHash
					storedExpiresAt = expiresAt
					return 7, nil
				},
			}

			err := newTestService(repo, emailSender).PreRegisterUser(context.Background(), PreRegisterInput{Email: " Ada@Example.COM "})
			if tt.wantErr == nil && err != nil {
				t.Fatalf("PreRegisterUser() error = %v", err)
			}
			if tt.wantErr != nil && err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != nil && tt.userErr == nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("PreRegisterUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantEmail {
				if storedEmail != "ada@example.com" {
					t.Fatalf("storedEmail = %q", storedEmail)
				}
				if len(emailSender.code) != 6 || strings.Trim(emailSender.code, "0123456789") != "" {
					t.Fatalf("verification code = %q, want six digits", emailSender.code)
				}
				if err := bcrypt.CompareHashAndPassword([]byte(storedCodeHash), []byte(emailSender.code)); err != nil {
					t.Fatalf("code hash mismatch: %v", err)
				}
				if !storedExpiresAt.Equal(newTestService(repo, emailSender).now().Add(verificationCodeTTL)) {
					t.Fatalf("storedExpiresAt = %v", storedExpiresAt)
				}
			}
		})
	}
}

func TestService_VerifyEmail(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	validHash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash code: %v", err)
	}

	tests := []struct {
		name       string
		repoResult PendingVerification
		repoErr    error
		inputCode  string
		wantErr    error
		wantToken  bool
	}{
		{
			name:       "happy path returns valid token",
			repoResult: PendingVerification{VerificationID: 9, Email: "ada@example.com", CodeHash: string(validHash), ExpiresAt: now.Add(time.Minute)},
			inputCode:  "123456",
			wantToken:  true,
		},
		{
			name:       "expired code",
			repoResult: PendingVerification{VerificationID: 9, Email: "ada@example.com", CodeHash: string(validHash), ExpiresAt: now.Add(-time.Minute)},
			inputCode:  "123456",
			wantErr:    ErrCodeExpired,
		},
		{
			name:       "invalid code",
			repoResult: PendingVerification{VerificationID: 9, Email: "ada@example.com", CodeHash: string(validHash), ExpiresAt: now.Add(time.Minute)},
			inputCode:  "000000",
			wantErr:    ErrCodeInvalid,
		},
		{
			name:      "verification not found",
			repoErr:   ErrVerificationNotFound,
			inputCode: "123456",
			wantErr:   ErrVerificationNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markedID int32
			repo := &mockUserRepository{
				getLatestPendingVerificationFunc: func(ctx context.Context, email string) (PendingVerification, error) {
					if email != "ada@example.com" {
						t.Fatalf("email = %q, want normalized ada@example.com", email)
					}
					return tt.repoResult, tt.repoErr
				},
				markPendingVerificationVerifiedFunc: func(ctx context.Context, verificationID int32) error {
					markedID = verificationID
					return nil
				},
			}
			svc := newTestService(repo, &mockEmailSender{})

			result, err := svc.VerifyEmail(context.Background(), VerifyEmailInput{Email: " Ada@Example.COM ", Code: tt.inputCode})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("VerifyEmail() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantToken {
				if result.VerificationToken == "" {
					t.Fatal("expected verification token")
				}
				emailValue, err := svc.verifyVerificationToken(result.VerificationToken)
				if err != nil {
					t.Fatalf("token should validate: %v", err)
				}
				if emailValue != "ada@example.com" {
					t.Fatalf("token email = %q", emailValue)
				}
				if markedID != 9 {
					t.Fatalf("markedID = %d, want 9", markedID)
				}
			}
		})
	}
}

func TestService_CompleteRegister(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		roleID     int32
		tokenSetup func(svc *service) string
		userErr    error
		wantErr    error
		wantRoleID int32
	}{
		{
			name:   "happy path with explicit role",
			roleID: 2,
			tokenSetup: func(svc *service) string {
				token, err := svc.generateVerificationToken(12, "ada@example.com", svc.now().Add(time.Minute))
				if err != nil {
					t.Fatalf("generate token: %v", err)
				}
				return token
			},
			wantRoleID: 2,
		},
		{
			name: "happy path defaults to client role",
			tokenSetup: func(svc *service) string {
				token, err := svc.generateVerificationToken(12, "ada@example.com", svc.now().Add(time.Minute))
				if err != nil {
					t.Fatalf("generate token: %v", err)
				}
				return token
			},
			wantRoleID: roleIDClient,
		},
		{
			name: "invalid token",
			tokenSetup: func(svc *service) string {
				return "invalid"
			},
			wantErr: ErrInvalidVerificationToken,
		},
		{
			name: "expired token",
			tokenSetup: func(svc *service) string {
				token, err := svc.generateVerificationToken(12, "ada@example.com", svc.now().Add(-time.Minute))
				if err != nil {
					t.Fatalf("generate token: %v", err)
				}
				return token
			},
			wantErr: ErrInvalidVerificationToken,
		},
		{
			name: "email already taken during completion",
			tokenSetup: func(svc *service) string {
				token, err := svc.generateVerificationToken(12, "ada@example.com", svc.now().Add(time.Minute))
				if err != nil {
					t.Fatalf("generate token: %v", err)
				}
				return token
			},
			userErr: ErrEmailTaken,
			wantErr: ErrEmailTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createdInput CreateUserRecord
			repo := &mockUserRepository{
				getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
					if tt.userErr != nil {
						return UserAuthRecord{Email: email}, nil
					}
					return UserAuthRecord{}, ErrUserNotFound
				},
				createUserFunc: func(ctx context.Context, input CreateUserRecord) (AuthUser, error) {
					createdInput = input
					return AuthUser{UserID: 10, UserUUID: input.UserUUID, Email: input.Email, RoleID: input.RoleID}, nil
				},
			}
			svc := newTestService(repo, &mockEmailSender{})
			token := tt.tokenSetup(svc)

			result, err := svc.CompleteRegister(ctx, CompleteRegisterInput{
				VerificationToken: token,
				FirstName:         "Ada",
				LastName:          "Lovelace",
				Password:          "supersecret",
				RoleID:            tt.roleID,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CompleteRegister() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if result.User.UserID != 10 {
				t.Fatalf("unexpected result: %+v", result)
			}
			if createdInput.Email != "ada@example.com" {
				t.Fatalf("created email = %q", createdInput.Email)
			}
			if createdInput.RoleID != tt.wantRoleID || createdInput.StatusID != statusIDActive {
				t.Fatalf("role/status = %d/%d", createdInput.RoleID, createdInput.StatusID)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(createdInput.PasswordHash), []byte("supersecret")); err != nil {
				t.Fatalf("password hash mismatch: %v", err)
			}
		})
	}
}

func TestLoginUserRequiresActiveAccountAndStoresHashedRefreshToken(t *testing.T) {
	ctx := context.Background()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	var storedHash string

	repo := &mockUserRepository{
		getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
			return UserAuthRecord{
				UserID:       5,
				UserUUID:     "8a6fbb17-b64b-4f40-a09d-b6639b357ef5",
				Email:        email,
				PasswordHash: string(passwordHash),
				RoleID:       roleIDClient,
				RoleName:     "Client",
				StatusID:     statusIDActive,
			}, nil
		},
		createRefreshTokenFunc: func(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error {
			storedHash = tokenHash
			return nil
		},
	}

	result, err := newTestService(repo, &mockEmailSender{}).LoginUser(ctx, LoginInput{
		Email:    "ada@example.com",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("LoginUser() error = %v", err)
	}
	if result.AccessToken != "access-token" || result.RefreshToken == "" {
		t.Fatalf("unexpected tokens: %+v", result)
	}
	if storedHash == "" || storedHash == result.RefreshToken || storedHash != hashRefreshToken(result.RefreshToken) {
		t.Fatalf("refresh token hash was not stored safely")
	}
}

func TestRefreshTokenRotatesStoredToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	oldToken := "old-refresh-token"
	var revokedHash string
	var newStoredHash string

	repo := &mockUserRepository{
		getRefreshTokenFunc: func(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
			if tokenHash != hashRefreshToken(oldToken) {
				t.Fatalf("tokenHash = %q", tokenHash)
			}
			return RefreshTokenRecord{UserID: 9, ExpiresAt: now.Add(time.Hour)}, nil
		},
		getUserByIDFunc: func(ctx context.Context, userID int32) (UserAuthRecord, error) {
			return UserAuthRecord{UserID: userID, UserUUID: "8a6fbb17-b64b-4f40-a09d-b6639b357ef5", RoleID: roleIDClient, RoleName: "Client"}, nil
		},
		revokeRefreshTokenFunc: func(ctx context.Context, tokenHash string) error {
			revokedHash = tokenHash
			return nil
		},
		createRefreshTokenFunc: func(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error {
			newStoredHash = tokenHash
			return nil
		},
	}
	svc := newTestService(repo, &mockEmailSender{})

	result, err := svc.RefreshToken(ctx, RefreshInput{RefreshToken: oldToken})
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if result.AccessToken != "access-token" || result.RefreshToken == "" {
		t.Fatalf("unexpected refresh result: %+v", result)
	}
	if revokedHash != hashRefreshToken(oldToken) {
		t.Fatalf("revokedHash = %q", revokedHash)
	}
	if newStoredHash == "" || newStoredHash == hashRefreshToken(oldToken) || newStoredHash != hashRefreshToken(result.RefreshToken) {
		t.Fatalf("new refresh token was not rotated")
	}
}

func TestDeleteUserSoftDeletesAndRevokesTokens(t *testing.T) {
	ctx := context.Background()
	userUUID := "8a6fbb17-b64b-4f40-a09d-b6639b357ef5"
	var softDeleted bool
	var revokedUserID int32

	repo := &mockUserRepository{
		getUserByUUIDFunc: func(ctx context.Context, uuidStr string) (UserAuthRecord, error) {
			if uuidStr != userUUID {
				t.Fatalf("uuidStr = %q", uuidStr)
			}
			return UserAuthRecord{UserID: 55, UserUUID: uuidStr}, nil
		},
		softDeleteUserFunc: func(ctx context.Context, uuidStr string) error {
			softDeleted = true
			return nil
		},
		revokeAllUserRefreshTokensFunc: func(ctx context.Context, userID int32) error {
			revokedUserID = userID
			return nil
		},
	}

	err := newTestService(repo, &mockEmailSender{}).DeleteUser(ctx, userUUID)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if !softDeleted || revokedUserID != 55 {
		t.Fatalf("softDeleted/revokedUserID = %v/%d", softDeleted, revokedUserID)
	}
}
