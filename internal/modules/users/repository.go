package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

type repository struct {
	queries *sqlcgen.Queries
}

func NewRepository(db *pgxpool.Pool) UserRepository {
	return &repository{queries: sqlcgen.New(db)}
}

func (r *repository) CreateUser(ctx context.Context, input CreateUserRecord) (AuthUser, error) {
	userUUID, err := toPgUUID(input.UserUUID)
	if err != nil {
		return AuthUser{}, fmt.Errorf("create user: %w", err)
	}

	user, err := r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		UserUuid:          userUUID,
		RoleID:            input.RoleID,
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Email:             input.Email,
		Password:          input.PasswordHash,
		Phone:             input.Phone,
		ProfilePictureUrl: input.ProfilePictureURL,
		StatusID:          input.StatusID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return AuthUser{}, ErrEmailTaken
		}
		return AuthUser{}, fmt.Errorf("create user: %w", err)
	}

	resolvedUUID, err := fromPgUUID(user.UserUuid)
	if err != nil {
		return AuthUser{}, fmt.Errorf("create user: %w", err)
	}

	return AuthUser{
		UserID:    user.UserID,
		UserUUID:  resolvedUUID,
		Email:     user.Email,
		RoleID:    input.RoleID,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (UserAuthRecord, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserAuthRecord{}, ErrUserNotFound
		}
		return UserAuthRecord{}, fmt.Errorf("get user by email: %w", err)
	}

	userUUID, err := fromPgUUID(user.UserUuid)
	if err != nil {
		return UserAuthRecord{}, fmt.Errorf("get user by email: %w", err)
	}

	return UserAuthRecord{
		UserID:       user.UserID,
		UserUUID:     userUUID,
		Email:        user.Email,
		PasswordHash: user.Password,
		RoleID:       user.RoleID,
		RoleName:     user.RoleName,
		StatusID:     user.StatusID,
		CreatedAt:    user.CreatedAt.Time,
	}, nil
}

func (r *repository) GetUserByUUID(ctx context.Context, uuidStr string) (UserAuthRecord, error) {
	userUUIDParam, err := toPgUUID(uuidStr)
	if err != nil {
		return UserAuthRecord{}, fmt.Errorf("get user by uuid: %w", err)
	}

	user, err := r.queries.GetUserByUUID(ctx, userUUIDParam)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserAuthRecord{}, ErrUserNotFound
		}
		return UserAuthRecord{}, fmt.Errorf("get user by uuid: %w", err)
	}

	userUUID, err := fromPgUUID(user.UserUuid)
	if err != nil {
		return UserAuthRecord{}, fmt.Errorf("get user by uuid: %w", err)
	}

	return UserAuthRecord{
		UserID:    user.UserID,
		UserUUID:  userUUID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		RoleName:  user.RoleName,
		StatusID:  user.StatusID,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}

func (r *repository) GetUserByID(ctx context.Context, userID int32) (UserAuthRecord, error) {
	user, err := r.queries.GetAuthenticatedUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserAuthRecord{}, ErrUserNotFound
		}
		return UserAuthRecord{}, fmt.Errorf("get user by id: %w", err)
	}

	userUUID, err := fromPgUUID(user.UserUuid)
	if err != nil {
		return UserAuthRecord{}, fmt.Errorf("get user by id: %w", err)
	}

	return UserAuthRecord{
		UserID:   user.UserID,
		UserUUID: userUUID,
		Email:    user.Email,
		RoleID:   user.RoleID,
		RoleName: user.RoleName,
	}, nil
}

func (r *repository) UpdateUserStatus(ctx context.Context, userID int32, statusID int32) error {
	if err := r.queries.UpdateUserStatus(ctx, sqlcgen.UpdateUserStatusParams{
		StatusID: statusID,
		UserID:   userID,
	}); err != nil {
		return fmt.Errorf("update user status: %w", err)
	}

	return nil
}

func (r *repository) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (AuthUser, error) {
	userUUID, err := toPgUUID(uuidStr)
	if err != nil {
		return AuthUser{}, fmt.Errorf("update profile: %w", err)
	}

	user, err := r.queries.UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		FirstName:         input.FirstName,
		LastName:          input.LastName,
		Phone:             input.Phone,
		ProfilePictureUrl: input.ProfilePictureURL,
		UserUuid:          userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthUser{}, ErrUserNotFound
		}
		return AuthUser{}, fmt.Errorf("update profile: %w", err)
	}

	resolvedUUID, err := fromPgUUID(user.UserUuid)
	if err != nil {
		return AuthUser{}, fmt.Errorf("update profile: %w", err)
	}

	return AuthUser{
		UserID:    user.UserID,
		UserUUID:  resolvedUUID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}

func (r *repository) SoftDeleteUser(ctx context.Context, uuidStr string) error {
	userUUID, err := toPgUUID(uuidStr)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	rowsAffected, err := r.queries.SoftDeleteUser(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *repository) CreatePendingVerification(ctx context.Context, email, codeHash string, expiresAt time.Time) (int32, error) {
	verification, err := r.queries.CreatePendingVerification(ctx, sqlcgen.CreatePendingVerificationParams{
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: toPgTimestamptz(expiresAt),
	})
	if err != nil {
		return 0, fmt.Errorf("create pending verification: %w", err)
	}

	return verification.VerificationID, nil
}

func (r *repository) GetLatestPendingVerification(ctx context.Context, email string) (PendingVerification, error) {
	verification, err := r.queries.GetLatestPendingVerification(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PendingVerification{}, ErrVerificationNotFound
		}
		return PendingVerification{}, fmt.Errorf("get latest pending verification: %w", err)
	}

	return PendingVerification{
		VerificationID: verification.VerificationID,
		Email:          verification.Email,
		CodeHash:       verification.CodeHash,
		ExpiresAt:      verification.ExpiresAt.Time,
		VerifiedAt:     nullableTime(verification.VerifiedAt),
	}, nil
}

func (r *repository) MarkPendingVerificationVerified(ctx context.Context, verificationID int32) error {
	if err := r.queries.MarkPendingVerificationVerified(ctx, verificationID); err != nil {
		return fmt.Errorf("mark pending verification verified: %w", err)
	}

	return nil
}

func (r *repository) CreateRefreshToken(ctx context.Context, userID int32, tokenHash string, expiresAt time.Time) error {
	if _, err := r.queries.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: toPgTimestamptz(expiresAt),
	}); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *repository) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshTokenRecord, error) {
	token, err := r.queries.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshTokenRecord{}, ErrInvalidCredentials
		}
		return RefreshTokenRecord{}, fmt.Errorf("get refresh token: %w", err)
	}

	return RefreshTokenRecord{
		UserID:    token.UserID,
		ExpiresAt: token.ExpiresAt.Time,
		RevokedAt: nullableTime(token.RevokedAt),
	}, nil
}

func (r *repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	rowsAffected, err := r.queries.RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if rowsAffected == 0 {
		return ErrInvalidCredentials
	}

	return nil
}

func (r *repository) RevokeAllUserRefreshTokens(ctx context.Context, userID int32) error {
	if _, err := r.queries.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke all user refresh tokens: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}

func toPgUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse uuid: %w", err)
	}

	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func fromPgUUID(value pgtype.UUID) (string, error) {
	if !value.Valid {
		return "", errors.New("uuid is null")
	}

	parsed, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return "", fmt.Errorf("format uuid: %w", err)
	}

	return parsed.String(), nil
}

func toPgTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}
