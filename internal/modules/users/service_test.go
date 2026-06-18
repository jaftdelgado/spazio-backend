package users

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	createUserFunc                         func(ctx context.Context, input CreateUserRecord) (AuthUser, error)
	getUserByEmailFunc                     func(ctx context.Context, email string) (UserAuthRecord, error)
	getUserByUUIDFunc                      func(ctx context.Context, uuidStr string) (UserAuthRecord, error)
	getUserByIDFunc                        func(ctx context.Context, userID int32) (UserAuthRecord, error)
	getUserProfileByUUIDFunc               func(ctx context.Context, uuidStr string) (UserProfile, error)
	listAgentsFunc                         func(ctx context.Context) ([]AgentListItem, error)
	updateUserStatusFunc                   func(ctx context.Context, userID int32, statusID int32) error
	updateProfileFunc                      func(ctx context.Context, uuidStr string, input UpdateProfileInput) (UserProfile, error)
	updateUserEmailFunc                    func(ctx context.Context, userID int32, email string) (UserProfile, error)
	updateUserPasswordFunc                 func(ctx context.Context, userID int32, passwordHash string) error
	updateUserProfilePhotoFunc             func(ctx context.Context, uuidStr string, profilePictureURL string) (UserProfile, error)
	softDeleteUserFunc                     func(ctx context.Context, uuidStr string) error
	createPendingVerificationFunc          func(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error)
	getLatestPendingVerificationFunc       func(ctx context.Context, email string) (PendingVerification, error)
	markPendingVerificationVerifiedFunc    func(ctx context.Context, verificationID int32) error
	createUserVerificationChallengeFunc    func(ctx context.Context, input CreateChallengeRecord) (int32, error)
	getLatestUserVerificationChallengeFunc func(ctx context.Context, email, purpose string) (UserVerificationChallenge, error)
	getUserVerificationChallengeByIDFunc   func(ctx context.Context, challengeID int32) (UserVerificationChallenge, error)
	markUserVerificationChallengeFunc      func(ctx context.Context, challengeID int32) error
	consumeUserVerificationChallengeFunc   func(ctx context.Context, challengeID int32) error
	createRefreshTokenFunc                 func(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error
	getRefreshTokenFunc                    func(ctx context.Context, tokenHash string) (RefreshTokenRecord, error)
	revokeRefreshTokenFunc                 func(ctx context.Context, tokenHash string) error
	revokeAllUserRefreshTokensFunc         func(ctx context.Context, userID int32) error
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
	return UserAuthRecord{}, ErrUserNotFound
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, userID int32) (UserAuthRecord, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, userID)
	}
	return UserAuthRecord{}, ErrUserNotFound
}

func (m *mockUserRepository) GetUserProfileByUUID(ctx context.Context, uuidStr string) (UserProfile, error) {
	if m.getUserProfileByUUIDFunc != nil {
		return m.getUserProfileByUUIDFunc(ctx, uuidStr)
	}
	return UserProfile{}, ErrUserNotFound
}

func (m *mockUserRepository) ListAgents(ctx context.Context) ([]AgentListItem, error) {
	if m.listAgentsFunc != nil {
		return m.listAgentsFunc(ctx)
	}
	return nil, nil
}

func (m *mockUserRepository) UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error {
	if m.updateUserStatusFunc != nil {
		return m.updateUserStatusFunc(ctx, userID, statusID)
	}
	return nil
}

func (m *mockUserRepository) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UserProfile, error) {
	if m.updateProfileFunc != nil {
		return m.updateProfileFunc(ctx, uuidStr, input)
	}
	return UserProfile{}, nil
}

func (m *mockUserRepository) UpdateUserEmail(ctx context.Context, userID int32, email string) (UserProfile, error) {
	if m.updateUserEmailFunc != nil {
		return m.updateUserEmailFunc(ctx, userID, email)
	}
	return UserProfile{}, nil
}

func (m *mockUserRepository) UpdateUserPassword(ctx context.Context, userID int32, passwordHash string) error {
	if m.updateUserPasswordFunc != nil {
		return m.updateUserPasswordFunc(ctx, userID, passwordHash)
	}
	return nil
}

func (m *mockUserRepository) UpdateUserProfilePhoto(ctx context.Context, uuidStr string, profilePictureURL string) (UserProfile, error) {
	if m.updateUserProfilePhotoFunc != nil {
		return m.updateUserProfilePhotoFunc(ctx, uuidStr, profilePictureURL)
	}
	return UserProfile{}, nil
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
	return PendingVerification{}, ErrVerificationNotFound
}

func (m *mockUserRepository) MarkPendingVerificationVerified(ctx context.Context, verificationID int32) error {
	if m.markPendingVerificationVerifiedFunc != nil {
		return m.markPendingVerificationVerifiedFunc(ctx, verificationID)
	}
	return nil
}

func (m *mockUserRepository) CreateUserVerificationChallenge(ctx context.Context, input CreateChallengeRecord) (int32, error) {
	if m.createUserVerificationChallengeFunc != nil {
		return m.createUserVerificationChallengeFunc(ctx, input)
	}
	return 1, nil
}

func (m *mockUserRepository) GetLatestUserVerificationChallenge(ctx context.Context, email, purpose string) (UserVerificationChallenge, error) {
	if m.getLatestUserVerificationChallengeFunc != nil {
		return m.getLatestUserVerificationChallengeFunc(ctx, email, purpose)
	}
	return UserVerificationChallenge{}, ErrVerificationNotFound
}

func (m *mockUserRepository) GetUserVerificationChallengeByID(ctx context.Context, challengeID int32) (UserVerificationChallenge, error) {
	if m.getUserVerificationChallengeByIDFunc != nil {
		return m.getUserVerificationChallengeByIDFunc(ctx, challengeID)
	}
	return UserVerificationChallenge{}, ErrVerificationNotFound
}

func (m *mockUserRepository) MarkUserVerificationChallengeVerified(ctx context.Context, challengeID int32) error {
	if m.markUserVerificationChallengeFunc != nil {
		return m.markUserVerificationChallengeFunc(ctx, challengeID)
	}
	return nil
}

func (m *mockUserRepository) ConsumeUserVerificationChallenge(ctx context.Context, challengeID int32) error {
	if m.consumeUserVerificationChallengeFunc != nil {
		return m.consumeUserVerificationChallengeFunc(ctx, challengeID)
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

type mockProfilePhotoStorage struct {
	uploadFunc    func(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	deleteFunc    func(ctx context.Context, storageKey string) error
	publicURLFunc func(ctx context.Context, storageKey string) (string, error)
}

func (m *mockProfilePhotoStorage) Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, storageKey, contentType, body)
	}
	return nil
}

func (m *mockProfilePhotoStorage) Delete(ctx context.Context, storageKey string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, storageKey)
	}
	return nil
}

func (m *mockProfilePhotoStorage) PublicURL(ctx context.Context, storageKey string) (string, error) {
	if m.publicURLFunc != nil {
		return m.publicURLFunc(ctx, storageKey)
	}
	return "https://cdn.example.com/" + storageKey, nil
}

func newTestService(repo UserRepository, emailSender *mockEmailSender, storage profilePhotoStorage) *service {
	svc := NewService(repo, emailSender, mockJWTService{token: "access-token"}, storage, "Spazio", "test-secret").(*service)
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
			repo := &mockUserRepository{
				getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
					if email != "ada@example.com" {
						t.Fatalf("email = %q", email)
					}
					if tt.userErr == nil {
						return UserAuthRecord{Email: email}, nil
					}
					return UserAuthRecord{}, tt.userErr
				},
				createPendingVerificationFunc: func(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error) {
					storedEmail = email
					storedCodeHash = codeHash
					return 7, nil
				},
			}

			err := newTestService(repo, emailSender, nil).PreRegisterUser(context.Background(), PreRegisterInput{Email: " Ada@Example.COM "})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("PreRegisterUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantEmail {
				if storedEmail != "ada@example.com" {
					t.Fatalf("storedEmail = %q", storedEmail)
				}
				if err := bcrypt.CompareHashAndPassword([]byte(storedCodeHash), []byte(emailSender.code)); err != nil {
					t.Fatalf("code hash mismatch: %v", err)
				}
			}
		})
	}
}

func TestService_VerifyPasswordResetCode(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	validHash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash code: %v", err)
	}
	userID := int32(9)

	tests := []struct {
		name      string
		challenge UserVerificationChallenge
		repoErr   error
		inputCode string
		wantErr   error
		wantToken bool
		markedID  int32
	}{
		{
			name:      "happy path returns reset token",
			challenge: UserVerificationChallenge{ChallengeID: 15, UserID: &userID, Email: "ada@example.com", Purpose: challengeResetPwd, CodeHash: string(validHash), ExpiresAt: now.Add(time.Minute)},
			inputCode: "123456",
			wantToken: true,
			markedID:  15,
		},
		{
			name:      "expired code",
			challenge: UserVerificationChallenge{ChallengeID: 15, UserID: &userID, Email: "ada@example.com", Purpose: challengeResetPwd, CodeHash: string(validHash), ExpiresAt: now.Add(-time.Minute)},
			inputCode: "123456",
			wantErr:   ErrCodeExpired,
		},
		{
			name:      "invalid code",
			challenge: UserVerificationChallenge{ChallengeID: 15, UserID: &userID, Email: "ada@example.com", Purpose: challengeResetPwd, CodeHash: string(validHash), ExpiresAt: now.Add(time.Minute)},
			inputCode: "000000",
			wantErr:   ErrCodeInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var markedID int32
			repo := &mockUserRepository{
				getLatestUserVerificationChallengeFunc: func(ctx context.Context, email, purpose string) (UserVerificationChallenge, error) {
					return tt.challenge, tt.repoErr
				},
				markUserVerificationChallengeFunc: func(ctx context.Context, challengeID int32) error {
					markedID = challengeID
					return nil
				},
			}
			svc := newTestService(repo, &mockEmailSender{}, nil)
			result, err := svc.VerifyPasswordResetCode(context.Background(), VerifyPasswordResetCodeInput{
				Email: "ada@example.com",
				Code:  tt.inputCode,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("VerifyPasswordResetCode() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantToken {
				if result.ResetToken == "" {
					t.Fatal("expected reset token")
				}
				if markedID != tt.markedID {
					t.Fatalf("markedID = %d, want %d", markedID, tt.markedID)
				}
			}
		})
	}
}

func TestService_ResetPassword(t *testing.T) {
	challengeUserID := int32(27)
	repo := &mockUserRepository{}
	svc := newTestService(repo, &mockEmailSender{}, nil)
	token, err := svc.generateActionToken(11, challengeUserID, "ada@example.com", challengeResetPwd, svc.now().Add(time.Minute))
	if err != nil {
		t.Fatalf("generateActionToken(): %v", err)
	}

	var updatedHash string
	var consumedID int32
	var revokedUserID int32
	repo.getUserVerificationChallengeByIDFunc = func(ctx context.Context, challengeID int32) (UserVerificationChallenge, error) {
		return UserVerificationChallenge{
			ChallengeID: challengeID,
			UserID:      &challengeUserID,
			Email:       "ada@example.com",
			Purpose:     challengeResetPwd,
			VerifiedAt:  timePtr(svc.now()),
		}, nil
	}
	repo.updateUserPasswordFunc = func(ctx context.Context, userID int32, passwordHash string) error {
		updatedHash = passwordHash
		return nil
	}
	repo.consumeUserVerificationChallengeFunc = func(ctx context.Context, challengeID int32) error {
		consumedID = challengeID
		return nil
	}
	repo.revokeAllUserRefreshTokensFunc = func(ctx context.Context, userID int32) error {
		revokedUserID = userID
		return nil
	}

	err = svc.ResetPassword(context.Background(), ResetPasswordInput{
		ResetToken:  token,
		NewPassword: "brandnewsecret",
	})
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte("brandnewsecret")); err != nil {
		t.Fatalf("password hash mismatch: %v", err)
	}
	if consumedID != 11 || revokedUserID != challengeUserID {
		t.Fatalf("consumedID/revokedUserID = %d/%d", consumedID, revokedUserID)
	}
}

func TestService_RequestEmailChange(t *testing.T) {
	userID := int32(5)
	var storedChallenge CreateChallengeRecord
	emailSender := &mockEmailSender{}
	repo := &mockUserRepository{
		getUserByUUIDFunc: func(ctx context.Context, uuidStr string) (UserAuthRecord, error) {
			return UserAuthRecord{UserID: userID, Email: "ada@example.com"}, nil
		},
		getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
			if email == "new@example.com" {
				return UserAuthRecord{}, ErrUserNotFound
			}
			return UserAuthRecord{UserID: 99}, nil
		},
		createUserVerificationChallengeFunc: func(ctx context.Context, input CreateChallengeRecord) (int32, error) {
			storedChallenge = input
			return 33, nil
		},
	}

	err := newTestService(repo, emailSender, nil).RequestEmailChange(context.Background(), "uuid-1", RequestEmailChangeInput{NewEmail: " New@Example.com "})
	if err != nil {
		t.Fatalf("RequestEmailChange() error = %v", err)
	}
	if storedChallenge.Purpose != challengeEmailChange || storedChallenge.Email != "new@example.com" || storedChallenge.UserID == nil || *storedChallenge.UserID != userID {
		t.Fatalf("unexpected stored challenge: %+v", storedChallenge)
	}
	if emailSender.to != "new@example.com" {
		t.Fatalf("emailSender.to = %q", emailSender.to)
	}
}

func TestService_ChangePassword(t *testing.T) {
	currentHash, err := bcrypt.GenerateFromPassword([]byte("current-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var updatedHash string
	var revokedUserID int32
	repo := &mockUserRepository{
		getUserByUUIDFunc: func(ctx context.Context, uuidStr string) (UserAuthRecord, error) {
			return UserAuthRecord{UserID: 8, PasswordHash: string(currentHash)}, nil
		},
		updateUserPasswordFunc: func(ctx context.Context, userID int32, passwordHash string) error {
			updatedHash = passwordHash
			return nil
		},
		revokeAllUserRefreshTokensFunc: func(ctx context.Context, userID int32) error {
			revokedUserID = userID
			return nil
		},
	}

	err = newTestService(repo, &mockEmailSender{}, nil).ChangePassword(context.Background(), "uuid-1", ChangePasswordInput{
		CurrentPassword: "current-secret",
		NewPassword:     "new-secret-123",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte("new-secret-123")); err != nil {
		t.Fatalf("new hash mismatch: %v", err)
	}
	if revokedUserID != 8 {
		t.Fatalf("revokedUserID = %d", revokedUserID)
	}
}

func TestService_UploadProfilePhoto(t *testing.T) {
	var uploadedKey string
	var uploadedType string
	var uploadedBytes []byte
	storage := &mockProfilePhotoStorage{
		uploadFunc: func(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
			uploadedKey = storageKey
			uploadedType = contentType
			data, _ := io.ReadAll(body)
			uploadedBytes = data
			return nil
		},
		publicURLFunc: func(ctx context.Context, storageKey string) (string, error) {
			return "https://cdn.example.com/" + storageKey, nil
		},
	}
	repo := &mockUserRepository{
		updateUserProfilePhotoFunc: func(ctx context.Context, uuidStr string, profilePictureURL string) (UserProfile, error) {
			return UserProfile{UserUUID: uuidStr, ProfilePictureURL: stringPtr(profilePictureURL)}, nil
		},
	}
	svc := newTestService(repo, &mockEmailSender{}, storage)
	svc.encodeToWebP = func(input UploadProfilePhotoInput) ([]byte, error) {
		return []byte("webp-bytes"), nil
	}

	result, err := svc.UploadProfilePhoto(context.Background(), UploadProfilePhotoInput{
		UserUUID: "user-uuid",
		File:     bytes.NewReader([]byte("raw-image")),
	})
	if err != nil {
		t.Fatalf("UploadProfilePhoto() error = %v", err)
	}
	if uploadedType != profilePhotoContentType || !strings.Contains(uploadedKey, "users/user-uuid/profile/") {
		t.Fatalf("unexpected upload key/type: %s %s", uploadedKey, uploadedType)
	}
	if string(uploadedBytes) != "webp-bytes" {
		t.Fatalf("uploadedBytes = %q", string(uploadedBytes))
	}
	if result.User.ProfilePictureURL == nil || !strings.Contains(*result.User.ProfilePictureURL, uploadedKey) {
		t.Fatalf("unexpected profile url: %+v", result.User.ProfilePictureURL)
	}
}

func TestService_AdminCreateUser(t *testing.T) {
	var createdInput CreateUserRecord
	repo := &mockUserRepository{
		getUserByEmailFunc: func(ctx context.Context, email string) (UserAuthRecord, error) {
			return UserAuthRecord{}, ErrUserNotFound
		},
		createUserFunc: func(ctx context.Context, input CreateUserRecord) (AuthUser, error) {
			createdInput = input
			return AuthUser{UserID: 14, UserUUID: input.UserUUID, Email: input.Email, RoleID: input.RoleID}, nil
		},
	}

	result, err := newTestService(repo, &mockEmailSender{}, nil).AdminCreateUser(context.Background(), AdminCreateUserInput{
		FirstName: "Grace",
		LastName:  "Hopper",
		Email:     "grace@example.com",
		Phone:     "555",
		RoleID:    roleIDAgent,
	})
	if err != nil {
		t.Fatalf("AdminCreateUser() error = %v", err)
	}
	if len(result.TemporaryPassword) != temporaryPasswordLength {
		t.Fatalf("temporary password length = %d", len(result.TemporaryPassword))
	}
	if createdInput.RoleID != roleIDAgent || createdInput.StatusID != statusIDActive {
		t.Fatalf("unexpected created input: %+v", createdInput)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(createdInput.PasswordHash), []byte(result.TemporaryPassword)); err != nil {
		t.Fatalf("temporary password hash mismatch: %v", err)
	}
}

func TestService_ListAgents(t *testing.T) {
	repo := &mockUserRepository{
		listAgentsFunc: func(ctx context.Context) ([]AgentListItem, error) {
			return []AgentListItem{{
				UserID:    21,
				UserUUID:  "agent-uuid",
				FirstName: "Ada",
				LastName:  "Lovelace",
			}}, nil
		},
	}

	result, err := newTestService(repo, &mockEmailSender{}, nil).ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].UserID != 21 {
		t.Fatalf("unexpected result: %+v", result)
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

	result, err := newTestService(repo, &mockEmailSender{}, nil).LoginUser(ctx, LoginInput{
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

	err := newTestService(repo, &mockEmailSender{}, nil).DeleteUser(ctx, userUUID)
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if !softDeleted || revokedUserID != 55 {
		t.Fatalf("softDeleted/revokedUserID = %v/%d", softDeleted, revokedUserID)
	}
}

func stringPtr(value string) *string {
	return &value
}
