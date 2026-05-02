package users

import (
	"context"
	"time"
)

type CreateUserInput struct {
	UserUUID          string `json:"user_uuid"`
	RoleID            int32  `json:"role_id"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Email             string `json:"email"`
	PasswordHash      string `json:"password_hash"`
	Phone             string `json:"phone"`
	ProfilePictureURL string `json:"profile_picture_url"`
	StatusID          int32  `json:"status_id"`
}

type CreateUserResult struct {
	UserID    int32     `json:"user_id"`
	UserUUID  string    `json:"user_uuid"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type VerifyUserInput struct {
	Email string `json:"email" binding:"required,email"`
	Token string `json:"token" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResult struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	User         CreateUserResult `json:"user"`
}

type UserRepository interface {
	CreateUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error)
	GetUserByEmail(ctx context.Context, email string) (CreateUserResult, error)
	UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error
}

type UserService interface {
	RegisterUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error)
	VerifyUser(ctx context.Context, email, token string) error
	LoginUser(ctx context.Context, input LoginInput) (LoginResult, error)
}
