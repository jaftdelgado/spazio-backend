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

type UserRepository interface {
	CreateUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error)
	GetUserByEmail(ctx context.Context, email string) (CreateUserResult, error)
}

type UserService interface {
	RegisterUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error)
}
