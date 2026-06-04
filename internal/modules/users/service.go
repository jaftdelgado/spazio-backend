package users

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"github.com/jaftdelgado/spazio-backend/internal/email"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost             = 12
	verificationCodeTTL    = 15 * time.Minute
	verificationTokenTTL   = 30 * time.Minute
	refreshTokenTTL        = 30 * 24 * time.Hour
	minPasswordLength      = 8
	registerSuccessMessage = "Cuenta creada correctamente."
)

type service struct {
	repository         UserRepository
	emailSender        email.EmailSender
	jwtService         auth.JWTService
	appName            string
	verificationSecret string
	now                func() time.Time
}

func NewService(repository UserRepository, emailSender email.EmailSender, jwtService auth.JWTService, appName, verificationSecret string) UserService {
	if appName == "" {
		appName = "Spazio"
	}

	return &service{
		repository:         repository,
		emailSender:        emailSender,
		jwtService:         jwtService,
		appName:            appName,
		verificationSecret: verificationSecret,
		now:                time.Now,
	}
}

func (s *service) PreRegisterUser(ctx context.Context, input PreRegisterInput) error {
	emailValue := normalizeEmail(input.Email)
	if _, err := s.repository.GetUserByEmail(ctx, emailValue); err == nil {
		return ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("check existing user: %w", err)
	}

	code, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("generate verification code: %w", err)
	}

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash verification code: %w", err)
	}

	if _, err := s.repository.CreatePendingVerification(ctx, emailValue, string(codeHash), s.now().Add(verificationCodeTTL)); err != nil {
		return fmt.Errorf("store pending verification: %w", err)
	}

	if err := s.emailSender.SendVerificationCode(emailValue, code, s.appName); err != nil {
		log.Printf("[DEV] verification code for %s: %s", emailValue, code)
	}

	return nil
}

func (s *service) VerifyEmail(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error) {
	emailValue := normalizeEmail(input.Email)
	verification, err := s.repository.GetLatestPendingVerification(ctx, emailValue)
	if err != nil {
		return VerifyEmailResult{}, fmt.Errorf("get pending verification: %w", err)
	}

	if verification.VerifiedAt != nil {
		return VerifyEmailResult{}, ErrEmailAlreadyVerified
	}
	if !verification.ExpiresAt.After(s.now()) {
		return VerifyEmailResult{}, ErrCodeExpired
	}
	if err := bcrypt.CompareHashAndPassword([]byte(verification.CodeHash), []byte(input.Code)); err != nil {
		return VerifyEmailResult{}, ErrCodeInvalid
	}

	if err := s.repository.MarkPendingVerificationVerified(ctx, verification.VerificationID); err != nil {
		return VerifyEmailResult{}, fmt.Errorf("mark pending verification verified: %w", err)
	}

	token, err := s.generateVerificationToken(verification.VerificationID, emailValue, s.now().Add(verificationTokenTTL))
	if err != nil {
		return VerifyEmailResult{}, fmt.Errorf("generate verification token: %w", err)
	}

	return VerifyEmailResult{VerificationToken: token}, nil
}

func (s *service) CompleteRegister(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error) {
	emailValue, err := s.verifyVerificationToken(input.VerificationToken)
	if err != nil {
		return RegisterResult{}, err
	}

	if _, err := s.repository.GetUserByEmail(ctx, emailValue); err == nil {
		return RegisterResult{}, ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return RegisterResult{}, fmt.Errorf("check existing user: %w", err)
	}

	if len(input.Password) < minPasswordLength {
		return RegisterResult{}, ErrInvalidCredentials
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("hash password: %w", err)
	}

	roleID := resolveRegisterRoleID(input.RoleID)
	user, err := s.repository.CreateUser(ctx, CreateUserRecord{
		UserUUID:          uuid.NewString(),
		RoleID:            roleID,
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Email:             emailValue,
		PasswordHash:      string(passwordHash),
		Phone:             input.Phone,
		ProfilePictureURL: input.ProfilePictureURL,
		StatusID:          statusIDActive,
	})
	if err != nil {
		return RegisterResult{}, fmt.Errorf("create user: %w", err)
	}

	return RegisterResult{
		Message: registerSuccessMessage,
		User:    user,
	}, nil
}

func (s *service) LoginUser(ctx context.Context, input LoginInput) (LoginResult, error) {
	user, err := s.repository.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("get user for login: %w", err)
	}

	if user.StatusID != statusIDActive {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	accessToken, err := s.jwtService.Generate(user.UserID, user.UserUUID, user.RoleID, user.RoleName)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	if err := s.repository.CreateRefreshToken(ctx, user.UserID, hashRefreshToken(refreshToken), s.now().Add(refreshTokenTTL)); err != nil {
		return LoginResult{}, fmt.Errorf("store refresh token: %w", err)
	}

	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: AuthUser{
			UserID:    user.UserID,
			UserUUID:  user.UserUUID,
			Email:     user.Email,
			RoleID:    user.RoleID,
			RoleName:  user.RoleName,
			CreatedAt: user.CreatedAt,
		},
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, input RefreshInput) (RefreshResult, error) {
	refreshHash := hashRefreshToken(input.RefreshToken)
	storedToken, err := s.repository.GetRefreshToken(ctx, refreshHash)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("get refresh token: %w", err)
	}
	if storedToken.RevokedAt != nil || !storedToken.ExpiresAt.After(s.now()) {
		return RefreshResult{}, ErrInvalidCredentials
	}

	user, err := s.repository.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return RefreshResult{}, ErrInvalidCredentials
		}
		return RefreshResult{}, fmt.Errorf("get refresh token user: %w", err)
	}

	if err := s.repository.RevokeRefreshToken(ctx, refreshHash); err != nil {
		return RefreshResult{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	accessToken, err := s.jwtService.Generate(user.UserID, user.UserUUID, user.RoleID, user.RoleName)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("generate access token: %w", err)
	}

	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return RefreshResult{}, fmt.Errorf("generate refresh token: %w", err)
	}
	if err := s.repository.CreateRefreshToken(ctx, user.UserID, hashRefreshToken(newRefreshToken), s.now().Add(refreshTokenTTL)); err != nil {
		return RefreshResult{}, fmt.Errorf("store refresh token: %w", err)
	}

	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *service) LogoutUser(ctx context.Context, input RefreshInput) error {
	if err := s.repository.RevokeRefreshToken(ctx, hashRefreshToken(input.RefreshToken)); err != nil {
		return fmt.Errorf("logout user: %w", err)
	}

	return nil
}

func (s *service) GetProfile(ctx context.Context, uuidStr string) (AuthUser, error) {
	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		return AuthUser{}, err
	}

	return AuthUser{
		UserID:    user.UserID,
		UserUUID:  user.UserUUID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		RoleName:  user.RoleName,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *service) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error) {
	user, err := s.repository.UpdateProfile(ctx, uuidStr, input)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("update profile: %w", err)
	}

	return UpdateProfileResult{
		Message: "Perfil actualizado correctamente.",
		User:    user,
	}, nil
}

func (s *service) DeleteUser(ctx context.Context, uuidStr string) error {
	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil || parsedUUID == uuid.Nil {
		return ErrUserNotFound
	}

	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("get user before delete: %w", err)
	}

	if err := s.repository.SoftDeleteUser(ctx, uuidStr); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if err := s.repository.RevokeAllUserRefreshTokens(ctx, user.UserID); err != nil {
		return fmt.Errorf("revoke deleted user refresh tokens: %w", err)
	}

	return nil
}

func generateVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", value.Int64()), nil
}

func generateRefreshToken() (string, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(token[:]), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func resolveRegisterRoleID(roleID int32) int32 {
	switch roleID {
	case 1, 2, 3:
		return roleID
	default:
		return roleIDClient
	}
}

func (s *service) generateVerificationToken(verificationID int32, emailValue string, expiresAt time.Time) (string, error) {
	if s.verificationSecret == "" {
		return "", errors.New("verification secret is not configured")
	}

	payload := fmt.Sprintf("%d:%s:%d", verificationID, emailValue, expiresAt.Unix())
	signature := signVerificationPayload(payload, s.verificationSecret)

	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *service) verifyVerificationToken(token string) (string, error) {
	if s.verificationSecret == "" {
		return "", ErrInvalidVerificationToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrInvalidVerificationToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidVerificationToken
	}
	gotSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidVerificationToken
	}

	payload := string(payloadBytes)
	wantSignature := signVerificationPayload(payload, s.verificationSecret)
	if !hmac.Equal(gotSignature, wantSignature) {
		return "", ErrInvalidVerificationToken
	}

	payloadParts := strings.Split(payload, ":")
	if len(payloadParts) != 3 {
		return "", ErrInvalidVerificationToken
	}
	if _, err := strconv.ParseInt(payloadParts[0], 10, 32); err != nil {
		return "", ErrInvalidVerificationToken
	}

	emailValue := normalizeEmail(payloadParts[1])
	if emailValue == "" {
		return "", ErrInvalidVerificationToken
	}

	expiresUnix, err := strconv.ParseInt(payloadParts[2], 10, 64)
	if err != nil {
		return "", ErrInvalidVerificationToken
	}
	if !time.Unix(expiresUnix, 0).After(s.now()) {
		return "", ErrInvalidVerificationToken
	}

	return emailValue, nil
}

func signVerificationPayload(payload, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}
