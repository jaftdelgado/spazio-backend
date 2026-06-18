package users

import (
	"context"
	"errors"
	"io"
	"time"
)

const (
	roleIDAdmin          int32 = 1
	roleIDAgent          int32 = 2
	roleIDClient         int32 = 3
	statusIDActive       int32 = 1
	challengeResetPwd          = "password_reset"
	challengeEmailChange       = "email_change"
)

var (
	ErrUserNotFound             = errors.New("user not found")
	ErrEmailTaken               = errors.New("email already taken")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrCodeExpired              = errors.New("verification code expired")
	ErrCodeInvalid              = errors.New("verification code invalid")
	ErrEmailAlreadyVerified     = errors.New("email already verified")
	ErrVerificationNotFound     = errors.New("verification not found")
	ErrInvalidVerificationToken = errors.New("invalid verification token")
	ErrInvalidActionToken       = errors.New("invalid action token")
	ErrCurrentPasswordInvalid   = errors.New("current password invalid")
	ErrInvalidRole              = errors.New("invalid role")
	ErrEmailUnchanged           = errors.New("email unchanged")
)

type PreRegisterInput struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyEmailInput struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// VerifyEmailResult is the response payload returned after email verification.
type VerifyEmailResult struct {
	VerificationToken string `json:"verification_token" example:"dGVzdEB..."`
}

type CompleteRegisterInput struct {
	VerificationToken string `json:"verification_token" binding:"required"`
	FirstName         string `json:"first_name" binding:"required"`
	LastName          string `json:"last_name" binding:"required"`
	Password          string `json:"password" binding:"required"`
	Phone             string `json:"phone"`
	ProfilePictureURL string `json:"profile_picture_url"`
	RoleID            int32  `json:"role_id"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}

type VerifyPasswordResetCodeInput struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

type ResetPasswordInput struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type UpdateProfileInput struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type RequestEmailChangeInput struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

type VerifyEmailChangeInput struct {
	NewEmail string `json:"new_email" binding:"required,email"`
	Code     string `json:"code" binding:"required"`
}

type ConfirmEmailChangeInput struct {
	VerificationToken string `json:"verification_token" binding:"required"`
}

type UploadProfilePhotoInput struct {
	UserUUID string
	File     io.Reader
}

type AdminCreateUserInput struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Phone     string `json:"phone"`
	RoleID    int32  `json:"role_id" binding:"required"`
}

// AgentListItem represents the lightweight agent payload used by property assignment UIs.
type AgentListItem struct {
	UserID            int32   `json:"user_id" example:"21"`
	UserUUID          string  `json:"user_uuid" example:"8b227e4e-ca58-41d9-b402-d773f95470ef"`
	FirstName         string  `json:"first_name" example:"Ada"`
	LastName          string  `json:"last_name" example:"Lovelace"`
	ProfilePictureURL *string `json:"profile_picture_url,omitempty" example:"https://cdn.example.com/users/profile.webp"`
}

// AuthUser represents an authenticated user returned by auth flows.
type AuthUser struct {
	UserID    int32     `json:"user_id" example:"13"`
	UserUUID  string    `json:"user_uuid" example:"8b227e4e-ca58-41d9-b402-d773f95470ef"`
	Email     string    `json:"email" example:"admin@spazio.com"`
	RoleID    int32     `json:"role_id" example:"1"`
	RoleName  string    `json:"role_name" example:"Admin"`
	CreatedAt time.Time `json:"created_at,omitempty" example:"2026-01-15T10:30:00Z"`
}

// UserProfile represents the editable profile data of a local user account.
type UserProfile struct {
	UserID            int32     `json:"user_id" example:"13"`
	UserUUID          string    `json:"user_uuid" example:"8b227e4e-ca58-41d9-b402-d773f95470ef"`
	RoleID            int32     `json:"role_id" example:"1"`
	RoleName          string    `json:"role_name" example:"Admin"`
	FirstName         string    `json:"first_name" example:"Ada"`
	LastName          string    `json:"last_name" example:"Lovelace"`
	Email             string    `json:"email" example:"ada@example.com"`
	Phone             string    `json:"phone,omitempty" example:"+525512345678"`
	ProfilePictureURL *string   `json:"profile_picture_url,omitempty" example:"https://cdn.example.com/users/profile.webp"`
	StatusID          int32     `json:"status_id" example:"1"`
	CreatedAt         time.Time `json:"created_at,omitempty" example:"2026-01-15T10:30:00Z"`
}

// RegisterResult is the response payload returned after successful registration.
type RegisterResult struct {
	Message string   `json:"message" example:"Cuenta creada correctamente."`
	User    AuthUser `json:"user"`
}

// LoginResult is the response payload returned after successful login.
type LoginResult struct {
	AccessToken  string   `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string   `json:"refresh_token" example:"4e9a58788701fce52001214130c15ac8..."`
	User         AuthUser `json:"user"`
}

// RefreshResult is the response payload returned after token rotation.
type RefreshResult struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"9f3b12788701fce52001214130c15ac8..."`
}

// PasswordResetVerificationResult is returned after validating a password reset code.
type PasswordResetVerificationResult struct {
	ResetToken string `json:"reset_token" example:"dGVzdA..."`
}

// EmailChangeVerificationResult is returned after validating an email change code.
type EmailChangeVerificationResult struct {
	VerificationToken string `json:"verification_token" example:"dGVzdA..."`
}

// UpdateProfileResult is the response payload returned after a profile update.
type UpdateProfileResult struct {
	Message string      `json:"message" example:"Perfil actualizado correctamente."`
	User    UserProfile `json:"user"`
}

// AdminCreateUserResult is returned after an admin creates a local staff account.
type AdminCreateUserResult struct {
	Message           string   `json:"message" example:"Usuario creado correctamente."`
	TemporaryPassword string   `json:"temporary_password" example:"X7m2Q9kL4pTz"`
	User              AuthUser `json:"user"`
}

// ListAgentsResult contains the lightweight staff list used to assign agents to properties.
type ListAgentsResult struct {
	Data []AgentListItem `json:"data"`
}

// MessageResult is a generic message response.
type MessageResult struct {
	Message string `json:"message" example:"Operación completada correctamente."`
}

type CreateUserRecord struct {
	UserUUID          string
	RoleID            int32
	FirstName         string
	LastName          string
	Email             string
	PasswordHash      string
	Phone             string
	ProfilePictureURL string
	StatusID          int32
}

type UserAuthRecord struct {
	UserID            int32
	UserUUID          string
	Email             string
	PasswordHash      string
	RoleID            int32
	RoleName          string
	StatusID          int32
	FirstName         string
	LastName          string
	Phone             string
	ProfilePictureURL *string
	CreatedAt         time.Time
}

type PendingVerification struct {
	VerificationID int32
	Email          string
	CodeHash       string
	ExpiresAt      time.Time
	VerifiedAt     *time.Time
}

type UserVerificationChallenge struct {
	ChallengeID int32
	UserID      *int32
	Email       string
	Purpose     string
	CodeHash    string
	ExpiresAt   time.Time
	VerifiedAt  *time.Time
	ConsumedAt  *time.Time
	CreatedAt   time.Time
}

type RefreshTokenRecord struct {
	UserID    int32
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type CreateChallengeRecord struct {
	UserID    *int32
	Email     string
	Purpose   string
	CodeHash  string
	ExpiresAt time.Time
}

type ActionTokenPayload struct {
	ChallengeID int32
	UserID      int32
	Email       string
	Purpose     string
}

type profilePhotoStorage interface {
	Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	Delete(ctx context.Context, storageKey string) error
	PublicURL(ctx context.Context, storageKey string) (string, error)
}

type UserRepository interface {
	CreateUser(ctx context.Context, input CreateUserRecord) (AuthUser, error)
	GetUserByEmail(ctx context.Context, email string) (UserAuthRecord, error)
	GetUserByUUID(ctx context.Context, uuidStr string) (UserAuthRecord, error)
	GetUserByID(ctx context.Context, userID int32) (UserAuthRecord, error)
	GetUserProfileByUUID(ctx context.Context, uuidStr string) (UserProfile, error)
	ListAgents(ctx context.Context) ([]AgentListItem, error)
	UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error
	UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UserProfile, error)
	UpdateUserEmail(ctx context.Context, userID int32, email string) (UserProfile, error)
	UpdateUserPassword(ctx context.Context, userID int32, passwordHash string) error
	UpdateUserProfilePhoto(ctx context.Context, uuidStr string, profilePictureURL string) (UserProfile, error)
	SoftDeleteUser(ctx context.Context, uuidStr string) error
	CreatePendingVerification(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error)
	GetLatestPendingVerification(ctx context.Context, email string) (PendingVerification, error)
	MarkPendingVerificationVerified(ctx context.Context, verificationID int32) error
	CreateUserVerificationChallenge(ctx context.Context, input CreateChallengeRecord) (int32, error)
	GetLatestUserVerificationChallenge(ctx context.Context, email, purpose string) (UserVerificationChallenge, error)
	GetUserVerificationChallengeByID(ctx context.Context, challengeID int32) (UserVerificationChallenge, error)
	MarkUserVerificationChallengeVerified(ctx context.Context, challengeID int32) error
	ConsumeUserVerificationChallenge(ctx context.Context, challengeID int32) error
	CreateRefreshToken(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int32) error
}

type UserService interface {
	PreRegisterUser(ctx context.Context, input PreRegisterInput) error
	VerifyEmail(ctx context.Context, input VerifyEmailInput) (VerifyEmailResult, error)
	CompleteRegister(ctx context.Context, input CompleteRegisterInput) (RegisterResult, error)
	LoginUser(ctx context.Context, input LoginInput) (LoginResult, error)
	RefreshToken(ctx context.Context, input RefreshInput) (RefreshResult, error)
	LogoutUser(ctx context.Context, input RefreshInput) error
	RequestPasswordReset(ctx context.Context, input ForgotPasswordInput) error
	VerifyPasswordResetCode(ctx context.Context, input VerifyPasswordResetCodeInput) (PasswordResetVerificationResult, error)
	ResetPassword(ctx context.Context, input ResetPasswordInput) error
	GetProfile(ctx context.Context, uuidStr string) (UserProfile, error)
	UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error)
	UploadProfilePhoto(ctx context.Context, input UploadProfilePhotoInput) (UpdateProfileResult, error)
	RequestEmailChange(ctx context.Context, uuidStr string, input RequestEmailChangeInput) error
	VerifyEmailChange(ctx context.Context, uuidStr string, input VerifyEmailChangeInput) (EmailChangeVerificationResult, error)
	ConfirmEmailChange(ctx context.Context, uuidStr string, input ConfirmEmailChangeInput) (UpdateProfileResult, error)
	ChangePassword(ctx context.Context, uuidStr string, input ChangePasswordInput) error
	ListAgents(ctx context.Context) (ListAgentsResult, error)
	AdminCreateUser(ctx context.Context, input AdminCreateUserInput) (AdminCreateUserResult, error)
	DeleteUser(ctx context.Context, uuidStr string) error
}
