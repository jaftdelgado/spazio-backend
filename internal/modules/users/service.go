package users

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/auth"
	"github.com/jaftdelgado/spazio-backend/internal/email"
	"github.com/nickalie/go-webpbin"
	"golang.org/x/crypto/bcrypt"
	_ "golang.org/x/image/webp"
)

const (
	bcryptCost                  = 12
	verificationCodeTTL         = 15 * time.Minute
	verificationTokenTTL        = 30 * time.Minute
	actionTokenTTL              = 30 * time.Minute
	refreshTokenTTL             = 30 * 24 * time.Hour
	minPasswordLength           = 8
	registerSuccessMessage      = "Cuenta creada correctamente."
	profileUpdatedMessage       = "Perfil actualizado correctamente."
	passwordUpdatedMessage      = "Contraseña actualizada correctamente."
	passwordResetSuccessMessage = "Contraseña restablecida correctamente."
	emailChangeRequestedMessage = "Código de verificación enviado al nuevo correo."
	emailChangedMessage         = "Correo actualizado correctamente."
	adminUserCreatedMessage     = "Usuario creado correctamente."
	profilePhotoContentType     = "image/webp"
	temporaryPasswordLength     = 12
)

type service struct {
	repository         UserRepository
	emailSender        email.EmailSender
	jwtService         auth.JWTService
	photoStorage       profilePhotoStorage
	appName            string
	verificationSecret string
	now                func() time.Time
	encodeToWebP       func(UploadProfilePhotoInput) ([]byte, error)
}

func NewService(repository UserRepository, emailSender email.EmailSender, jwtService auth.JWTService, photoStorage profilePhotoStorage, appName, verificationSecret string) UserService {
	if appName == "" {
		appName = "Spazio"
	}

	return &service{
		repository:         repository,
		emailSender:        emailSender,
		jwtService:         jwtService,
		photoStorage:       photoStorage,
		appName:            appName,
		verificationSecret: verificationSecret,
		now:                time.Now,
		encodeToWebP:       convertProfilePhotoToWebP,
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

	_ = s.emailSender.SendVerificationCode(emailValue, code, s.appName)

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
	user, err := s.repository.GetUserByEmail(ctx, normalizeEmail(input.Email))
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

func (s *service) RequestPasswordReset(ctx context.Context, input ForgotPasswordInput) error {
	emailValue := normalizeEmail(input.Email)
	user, err := s.repository.GetUserByEmail(ctx, emailValue)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("get user for password reset: %w", err)
	}
	if user.StatusID != statusIDActive {
		return nil
	}

	return s.createAndSendChallenge(ctx, CreateChallengeRecord{
		UserID:    &user.UserID,
		Email:     emailValue,
		Purpose:   challengeResetPwd,
		ExpiresAt: s.now().Add(verificationCodeTTL),
	})
}

func (s *service) VerifyPasswordResetCode(ctx context.Context, input VerifyPasswordResetCodeInput) (PasswordResetVerificationResult, error) {
	challenge, err := s.verifyChallengeCode(ctx, normalizeEmail(input.Email), challengeResetPwd, input.Code)
	if err != nil {
		return PasswordResetVerificationResult{}, err
	}

	userID := int32(0)
	if challenge.UserID != nil {
		userID = *challenge.UserID
	}

	token, err := s.generateActionToken(challenge.ChallengeID, userID, challenge.Email, challenge.Purpose, s.now().Add(actionTokenTTL))
	if err != nil {
		return PasswordResetVerificationResult{}, fmt.Errorf("generate reset token: %w", err)
	}

	return PasswordResetVerificationResult{ResetToken: token}, nil
}

func (s *service) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	if len(input.NewPassword) < minPasswordLength {
		return ErrInvalidCredentials
	}

	payload, err := s.verifyActionToken(input.ResetToken, challengeResetPwd)
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := s.repository.UpdateUserPassword(ctx, payload.UserID, string(passwordHash)); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if err := s.repository.ConsumeUserVerificationChallenge(ctx, payload.ChallengeID); err != nil {
		return fmt.Errorf("consume password reset challenge: %w", err)
	}
	if err := s.repository.RevokeAllUserRefreshTokens(ctx, payload.UserID); err != nil {
		return fmt.Errorf("revoke refresh tokens after reset: %w", err)
	}

	return nil
}

func (s *service) GetProfile(ctx context.Context, uuidStr string) (UserProfile, error) {
	user, err := s.repository.GetUserProfileByUUID(ctx, uuidStr)
	if err != nil {
		return UserProfile{}, err
	}

	return user, nil
}

func (s *service) ListAgents(ctx context.Context) (ListAgentsResult, error) {
	agents, err := s.repository.ListAgents(ctx)
	if err != nil {
		return ListAgentsResult{}, fmt.Errorf("list agents: %w", err)
	}

	return ListAgentsResult{Data: agents}, nil
}

func (s *service) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error) {
	user, err := s.repository.UpdateProfile(ctx, uuidStr, input)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("update profile: %w", err)
	}

	return UpdateProfileResult{
		Message: profileUpdatedMessage,
		User:    user,
	}, nil
}

func (s *service) UploadProfilePhoto(ctx context.Context, input UploadProfilePhotoInput) (UpdateProfileResult, error) {
	if s.photoStorage == nil {
		return UpdateProfileResult{}, errors.New("profile photo storage is not configured")
	}

	webpData, err := s.encodeToWebP(input)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("convert profile photo to webp: %w", err)
	}

	storageKey := fmt.Sprintf("users/%s/profile/%s.webp", input.UserUUID, uuid.NewString())
	if err := s.photoStorage.Upload(ctx, storageKey, profilePhotoContentType, bytes.NewReader(webpData)); err != nil {
		return UpdateProfileResult{}, fmt.Errorf("upload profile photo: %w", err)
	}

	publicURL, err := s.photoStorage.PublicURL(ctx, storageKey)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("resolve profile photo public url: %w", err)
	}

	user, err := s.repository.UpdateUserProfilePhoto(ctx, input.UserUUID, publicURL)
	if err != nil {
		_ = s.photoStorage.Delete(ctx, storageKey)
		return UpdateProfileResult{}, fmt.Errorf("update profile photo url: %w", err)
	}

	return UpdateProfileResult{
		Message: profileUpdatedMessage,
		User:    user,
	}, nil
}

func (s *service) RequestEmailChange(ctx context.Context, uuidStr string, input RequestEmailChangeInput) error {
	newEmail := normalizeEmail(input.NewEmail)
	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		return fmt.Errorf("get user for email change: %w", err)
	}
	if normalizeEmail(user.Email) == newEmail {
		return ErrEmailUnchanged
	}

	if existing, err := s.repository.GetUserByEmail(ctx, newEmail); err == nil && existing.UserID != user.UserID {
		return ErrEmailTaken
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("check target email: %w", err)
	}

	return s.createAndSendChallenge(ctx, CreateChallengeRecord{
		UserID:    &user.UserID,
		Email:     newEmail,
		Purpose:   challengeEmailChange,
		ExpiresAt: s.now().Add(verificationCodeTTL),
	})
}

func (s *service) VerifyEmailChange(ctx context.Context, uuidStr string, input VerifyEmailChangeInput) (EmailChangeVerificationResult, error) {
	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		return EmailChangeVerificationResult{}, fmt.Errorf("get user for email change verification: %w", err)
	}

	challenge, err := s.verifyChallengeCode(ctx, normalizeEmail(input.NewEmail), challengeEmailChange, input.Code)
	if err != nil {
		return EmailChangeVerificationResult{}, err
	}
	if challenge.UserID == nil || *challenge.UserID != user.UserID {
		return EmailChangeVerificationResult{}, ErrVerificationNotFound
	}

	token, err := s.generateActionToken(challenge.ChallengeID, user.UserID, challenge.Email, challenge.Purpose, s.now().Add(actionTokenTTL))
	if err != nil {
		return EmailChangeVerificationResult{}, fmt.Errorf("generate email change token: %w", err)
	}

	return EmailChangeVerificationResult{VerificationToken: token}, nil
}

func (s *service) ConfirmEmailChange(ctx context.Context, uuidStr string, input ConfirmEmailChangeInput) (UpdateProfileResult, error) {
	payload, err := s.verifyActionToken(input.VerificationToken, challengeEmailChange)
	if err != nil {
		return UpdateProfileResult{}, err
	}

	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("get user for confirm email change: %w", err)
	}
	if user.UserID != payload.UserID {
		return UpdateProfileResult{}, ErrInvalidActionToken
	}

	if existing, err := s.repository.GetUserByEmail(ctx, payload.Email); err == nil && existing.UserID != user.UserID {
		return UpdateProfileResult{}, ErrEmailTaken
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return UpdateProfileResult{}, fmt.Errorf("check target email before update: %w", err)
	}

	updatedProfile, err := s.repository.UpdateUserEmail(ctx, payload.UserID, payload.Email)
	if err != nil {
		return UpdateProfileResult{}, fmt.Errorf("update user email: %w", err)
	}
	if err := s.repository.ConsumeUserVerificationChallenge(ctx, payload.ChallengeID); err != nil {
		return UpdateProfileResult{}, fmt.Errorf("consume email change challenge: %w", err)
	}

	return UpdateProfileResult{
		Message: emailChangedMessage,
		User:    updatedProfile,
	}, nil
}

func (s *service) ChangePassword(ctx context.Context, uuidStr string, input ChangePasswordInput) error {
	if len(input.NewPassword) < minPasswordLength {
		return ErrInvalidCredentials
	}

	user, err := s.repository.GetUserByUUID(ctx, uuidStr)
	if err != nil {
		return fmt.Errorf("get user for password change: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.CurrentPassword)); err != nil {
		return ErrCurrentPasswordInvalid
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash changed password: %w", err)
	}
	if err := s.repository.UpdateUserPassword(ctx, user.UserID, string(passwordHash)); err != nil {
		return fmt.Errorf("update changed password: %w", err)
	}
	if err := s.repository.RevokeAllUserRefreshTokens(ctx, user.UserID); err != nil {
		return fmt.Errorf("revoke refresh tokens after password change: %w", err)
	}

	return nil
}

func (s *service) AdminCreateUser(ctx context.Context, input AdminCreateUserInput) (AdminCreateUserResult, error) {
	if input.RoleID != roleIDAdmin && input.RoleID != roleIDAgent {
		return AdminCreateUserResult{}, ErrInvalidRole
	}

	emailValue := normalizeEmail(input.Email)
	if _, err := s.repository.GetUserByEmail(ctx, emailValue); err == nil {
		return AdminCreateUserResult{}, ErrEmailTaken
	} else if !errors.Is(err, ErrUserNotFound) {
		return AdminCreateUserResult{}, fmt.Errorf("check staff email: %w", err)
	}

	temporaryPassword, err := generateTemporaryPassword(temporaryPasswordLength)
	if err != nil {
		return AdminCreateUserResult{}, fmt.Errorf("generate temporary password: %w", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcryptCost)
	if err != nil {
		return AdminCreateUserResult{}, fmt.Errorf("hash temporary password: %w", err)
	}

	user, err := s.repository.CreateUser(ctx, CreateUserRecord{
		UserUUID:          uuid.NewString(),
		RoleID:            input.RoleID,
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Email:             emailValue,
		PasswordHash:      string(passwordHash),
		Phone:             input.Phone,
		ProfilePictureURL: "",
		StatusID:          statusIDActive,
	})
	if err != nil {
		return AdminCreateUserResult{}, fmt.Errorf("create admin user: %w", err)
	}

	return AdminCreateUserResult{
		Message:           adminUserCreatedMessage,
		TemporaryPassword: temporaryPassword,
		User:              user,
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

func (s *service) createAndSendChallenge(ctx context.Context, input CreateChallengeRecord) error {
	code, err := generateVerificationCode()
	if err != nil {
		return fmt.Errorf("generate verification code: %w", err)
	}

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash verification code: %w", err)
	}

	if _, err := s.repository.CreateUserVerificationChallenge(ctx, CreateChallengeRecord{
		UserID:    input.UserID,
		Email:     input.Email,
		Purpose:   input.Purpose,
		CodeHash:  string(codeHash),
		ExpiresAt: input.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("store user verification challenge: %w", err)
	}

	_ = s.emailSender.SendVerificationCode(input.Email, code, s.appName)

	return nil
}

func (s *service) verifyChallengeCode(ctx context.Context, emailValue, purpose, code string) (UserVerificationChallenge, error) {
	challenge, err := s.repository.GetLatestUserVerificationChallenge(ctx, emailValue, purpose)
	if err != nil {
		return UserVerificationChallenge{}, fmt.Errorf("get user verification challenge: %w", err)
	}
	if challenge.ConsumedAt != nil || challenge.VerifiedAt != nil {
		return UserVerificationChallenge{}, ErrCodeInvalid
	}
	if !challenge.ExpiresAt.After(s.now()) {
		return UserVerificationChallenge{}, ErrCodeExpired
	}
	if err := bcrypt.CompareHashAndPassword([]byte(challenge.CodeHash), []byte(code)); err != nil {
		return UserVerificationChallenge{}, ErrCodeInvalid
	}
	if err := s.repository.MarkUserVerificationChallengeVerified(ctx, challenge.ChallengeID); err != nil {
		return UserVerificationChallenge{}, fmt.Errorf("mark user verification challenge verified: %w", err)
	}

	challenge.VerifiedAt = timePtr(s.now())
	return challenge, nil
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

func generateTemporaryPassword(length int) (string, error) {
	if length < minPasswordLength {
		length = minPasswordLength
	}

	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	bytes := make([]byte, length)
	for i := range bytes {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		bytes[i] = alphabet[n.Int64()]
	}

	return string(bytes), nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func resolveRegisterRoleID(roleID int32) int32 {
	switch roleID {
	case roleIDAdmin, roleIDAgent, roleIDClient:
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

func (s *service) generateActionToken(challengeID, userID int32, emailValue, purpose string, expiresAt time.Time) (string, error) {
	if s.verificationSecret == "" {
		return "", errors.New("verification secret is not configured")
	}

	payload := fmt.Sprintf("%d:%d:%s:%s:%d", challengeID, userID, emailValue, purpose, expiresAt.Unix())
	signature := signVerificationPayload(payload, s.verificationSecret)

	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *service) verifyActionToken(token string, expectedPurpose string) (ActionTokenPayload, error) {
	if s.verificationSecret == "" {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}
	gotSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	payload := string(payloadBytes)
	wantSignature := signVerificationPayload(payload, s.verificationSecret)
	if !hmac.Equal(gotSignature, wantSignature) {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	payloadParts := strings.Split(payload, ":")
	if len(payloadParts) != 5 {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	challengeID, err := strconv.ParseInt(payloadParts[0], 10, 32)
	if err != nil {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}
	userID, err := strconv.ParseInt(payloadParts[1], 10, 32)
	if err != nil || userID <= 0 {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	emailValue := normalizeEmail(payloadParts[2])
	if emailValue == "" {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	purpose := payloadParts[3]
	if purpose != expectedPurpose {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	expiresUnix, err := strconv.ParseInt(payloadParts[4], 10, 64)
	if err != nil || !time.Unix(expiresUnix, 0).After(s.now()) {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	challenge, err := s.repository.GetUserVerificationChallengeByID(context.Background(), int32(challengeID))
	if err != nil {
		return ActionTokenPayload{}, fmt.Errorf("get verification challenge by id: %w", err)
	}
	if challenge.ConsumedAt != nil || challenge.VerifiedAt == nil || challenge.Purpose != expectedPurpose || normalizeEmail(challenge.Email) != emailValue {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}
	if challenge.UserID == nil || *challenge.UserID != int32(userID) {
		return ActionTokenPayload{}, ErrInvalidActionToken
	}

	return ActionTokenPayload{
		ChallengeID: int32(challengeID),
		UserID:      int32(userID),
		Email:       emailValue,
		Purpose:     purpose,
	}, nil
}

func signVerificationPayload(payload, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func convertProfilePhotoToWebP(input UploadProfilePhotoInput) ([]byte, error) {
	img, _, err := image.Decode(input.File)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	var buf bytes.Buffer
	if err := webpbin.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode webp: %w", err)
	}

	return buf.Bytes(), nil
}

func timePtr(value time.Time) *time.Time {
	return &value
}
