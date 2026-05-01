package users

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) UserRepository {
	return &repository{db: db}
}

func (r *repository) CreateUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error) {
	queries := sqlcgen.New(r.db)

	var userUUID pgtype.UUID
	if err := userUUID.Scan(input.UserUUID); err != nil {
		return CreateUserResult{}, fmt.Errorf("invalid uuid format: %w", err)
	}

	params := sqlcgen.CreateUserParams{
		UserUuid:          userUUID,
		RoleID:            input.RoleID,
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Email:             input.Email,
		Phone:             input.Phone,
		ProfilePictureUrl: input.ProfilePictureURL,
		StatusID:          input.StatusID,
	}

	user, err := queries.CreateUser(ctx, params)
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("error creating user in db: %w", err)
	}

	return CreateUserResult{
		UserID:    user.UserID,
		UserUUID:  fmt.Sprintf("%x-%x-%x-%x-%x", user.UserUuid.Bytes[0:4], user.UserUuid.Bytes[4:6], user.UserUuid.Bytes[6:8], user.UserUuid.Bytes[8:10], user.UserUuid.Bytes[10:16]),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (CreateUserResult, error) {
	queries := sqlcgen.New(r.db)

	user, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("error getting user by email: %w", err)
	}

	return CreateUserResult{
		UserID:    user.UserID,
		UserUUID:  fmt.Sprintf("%x-%x-%x-%x-%x", user.UserUuid.Bytes[0:4], user.UserUuid.Bytes[4:6], user.UserUuid.Bytes[6:8], user.UserUuid.Bytes[8:10], user.UserUuid.Bytes[10:16]),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}
