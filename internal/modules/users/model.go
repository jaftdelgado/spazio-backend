package users

import (
	"context"
	"errors"
	"time"
)

const (
	roleIDClient    int32 = 3
	statusIDPending int32 = 3
	statusIDActive  int32 = 1
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

type UpdateProfileInput struct {
	FirstName         string `json:"first_name" binding:"required"`
	LastName          string `json:"last_name" binding:"required"`
	Phone             string `json:"phone"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

// AuthUser represents an authenticated user returned by the API.
type AuthUser struct {
	UserID    int32     `json:"user_id" example:"13"`
	UserUUID  string    `json:"user_uuid" example:"8b227e4e-ca58-41d9-b402-d773f95470ef"`
	Email     string    `json:"email" example:"admin@spazio.com"`
	RoleID    int32     `json:"role_id" example:"1"`
	RoleName  string    `json:"role_name" example:"Admin"`
	CreatedAt time.Time `json:"created_at,omitempty" example:"2026-01-15T10:30:00Z"`
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

// UpdateProfileResult is the response payload returned after a profile update.
type UpdateProfileResult struct {
	Message string   `json:"message" example:"Perfil actualizado correctamente."`
	User    AuthUser `json:"user"`
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
	UserID       int32
	UserUUID     string
	Email        string
	PasswordHash string
	RoleID       int32
	RoleName     string
	StatusID     int32
	CreatedAt    time.Time
}

type PendingVerification struct {
	VerificationID int32
	Email          string
	CodeHash       string
	ExpiresAt      time.Time
	VerifiedAt     *time.Time
}

type RefreshTokenRecord struct {
	UserID    int32
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type UserRepository interface {
	CreateUser(ctx context.Context, input CreateUserRecord) (AuthUser, error)
	GetUserByEmail(ctx context.Context, email string) (UserAuthRecord, error)
	GetUserByUUID(ctx context.Context, uuidStr string) (UserAuthRecord, error)
	GetUserByID(ctx context.Context, userID int32) (UserAuthRecord, error)
	UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error
	UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (AuthUser, error)
	SoftDeleteUser(ctx context.Context, uuidStr string) error
	CreatePendingVerification(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error)
	GetLatestPendingVerification(ctx context.Context, email string) (PendingVerification, error)
	MarkPendingVerificationVerified(ctx context.Context, verificationID int32) error
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
	GetProfile(ctx context.Context, uuidStr string) (AuthUser, error)
	UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UpdateProfileResult, error)
	DeleteUser(ctx context.Context, uuidStr string) error
}
