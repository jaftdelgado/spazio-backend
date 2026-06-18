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
		ProfilePictureUrl: pgtype.Text{String: input.ProfilePictureURL, Valid: input.ProfilePictureURL != ""},
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
		UserID:            user.UserID,
		UserUUID:          userUUID,
		Email:             user.Email,
		PasswordHash:      user.Password,
		RoleID:            user.RoleID,
		RoleName:          user.RoleName,
		StatusID:          user.StatusID,
		FirstName:         user.FirstName,
		LastName:          user.LastName,
		Phone:             user.Phone,
		ProfilePictureURL: nullableString(user.ProfilePictureUrl),
		CreatedAt:         user.CreatedAt.Time,
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

func (r *repository) GetUserProfileByUUID(ctx context.Context, uuidStr string) (UserProfile, error) {
	userUUID, err := toPgUUID(uuidStr)
	if err != nil {
		return UserProfile{}, fmt.Errorf("get user profile by uuid: %w", err)
	}

	user, err := r.queries.GetUserProfileByUUID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, fmt.Errorf("get user profile by uuid: %w", err)
	}

	return toUserProfile(
		user.UserID,
		user.UserUuid,
		user.Email,
		user.RoleID,
		user.RoleName,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.ProfilePictureUrl,
		user.StatusID,
		user.CreatedAt,
	)
}

func (r *repository) ListAgents(ctx context.Context) ([]AgentListItem, error) {
	rows, err := r.queries.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	agents := make([]AgentListItem, 0, len(rows))
	for _, row := range rows {
		userUUID, err := fromPgUUID(row.UserUuid)
		if err != nil {
			return nil, fmt.Errorf("list agents: %w", err)
		}

		agents = append(agents, AgentListItem{
			UserID:            row.UserID,
			UserUUID:          userUUID,
			FirstName:         row.FirstName,
			LastName:          row.LastName,
			ProfilePictureURL: nullableString(row.ProfilePictureUrl),
		})
	}

	return agents, nil
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

func (r *repository) UpdateProfile(ctx context.Context, uuidStr string, input UpdateProfileInput) (UserProfile, error) {
	userUUID, err := toPgUUID(uuidStr)
	if err != nil {
		return UserProfile{}, fmt.Errorf("update profile: %w", err)
	}

	if _, err := r.queries.UpdateUserProfile(ctx, sqlcgen.UpdateUserProfileParams{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Phone:     input.Phone,
		UserUuid:  userUUID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, fmt.Errorf("update profile: %w", err)
	}

	profile, err := r.GetUserProfileByUUID(ctx, uuidStr)
	if err != nil {
		return UserProfile{}, err
	}

	return profile, nil
}

func (r *repository) UpdateUserEmail(ctx context.Context, userID int32, email string) (UserProfile, error) {
	user, err := r.queries.UpdateUserEmail(ctx, sqlcgen.UpdateUserEmailParams{
		Email:  email,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserProfile{}, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return UserProfile{}, ErrEmailTaken
		}
		return UserProfile{}, fmt.Errorf("update user email: %w", err)
	}

	getUser, err := r.queries.GetUserByUUID(ctx, user.UserUuid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, fmt.Errorf("load user after email update: %w", err)
	}

	return toUserProfile(
		getUser.UserID,
		getUser.UserUuid,
		getUser.Email,
		getUser.RoleID,
		getUser.RoleName,
		getUser.FirstName,
		getUser.LastName,
		getUser.Phone,
		getUser.ProfilePictureUrl,
		getUser.StatusID,
		getUser.CreatedAt,
	)
}

func (r *repository) UpdateUserPassword(ctx context.Context, userID int32, passwordHash string) error {
	rowsAffected, err := r.queries.UpdateUserPassword(ctx, sqlcgen.UpdateUserPasswordParams{
		Password: passwordHash,
		UserID:   userID,
	})
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *repository) UpdateUserProfilePhoto(ctx context.Context, uuidStr string, profilePictureURL string) (UserProfile, error) {
	userUUID, err := toPgUUID(uuidStr)
	if err != nil {
		return UserProfile{}, fmt.Errorf("update user profile photo: %w", err)
	}

	if _, err := r.queries.UpdateUserProfilePhoto(ctx, sqlcgen.UpdateUserProfilePhotoParams{
		ProfilePictureUrl: pgtype.Text{String: profilePictureURL, Valid: profilePictureURL != ""},
		UserUuid:          userUUID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserProfile{}, ErrUserNotFound
		}
		return UserProfile{}, fmt.Errorf("update user profile photo: %w", err)
	}

	profile, err := r.GetUserProfileByUUID(ctx, uuidStr)
	if err != nil {
		return UserProfile{}, err
	}

	return profile, nil
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

func (r *repository) CreateUserVerificationChallenge(ctx context.Context, input CreateChallengeRecord) (int32, error) {
	arg := sqlcgen.CreateUserVerificationChallengeParams{
		Email:     input.Email,
		Purpose:   input.Purpose,
		CodeHash:  input.CodeHash,
		ExpiresAt: toPgTimestamptz(input.ExpiresAt),
	}
	if input.UserID != nil {
		arg.UserID = pgtype.Int4{Int32: *input.UserID, Valid: true}
	}

	row, err := r.queries.CreateUserVerificationChallenge(ctx, arg)
	if err != nil {
		return 0, fmt.Errorf("create user verification challenge: %w", err)
	}

	return row.ChallengeID, nil
}

func (r *repository) GetLatestUserVerificationChallenge(ctx context.Context, email, purpose string) (UserVerificationChallenge, error) {
	row, err := r.queries.GetLatestUserVerificationChallenge(ctx, sqlcgen.GetLatestUserVerificationChallengeParams{
		Email:   email,
		Purpose: purpose,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserVerificationChallenge{}, ErrVerificationNotFound
		}
		return UserVerificationChallenge{}, fmt.Errorf("get latest user verification challenge: %w", err)
	}

	return toUserVerificationChallenge(row.ChallengeID, row.UserID, row.Email, row.Purpose, row.CodeHash, row.ExpiresAt, row.VerifiedAt, row.ConsumedAt, row.CreatedAt), nil
}

func (r *repository) GetUserVerificationChallengeByID(ctx context.Context, challengeID int32) (UserVerificationChallenge, error) {
	row, err := r.queries.GetUserVerificationChallengeByID(ctx, challengeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserVerificationChallenge{}, ErrVerificationNotFound
		}
		return UserVerificationChallenge{}, fmt.Errorf("get user verification challenge by id: %w", err)
	}

	return toUserVerificationChallenge(row.ChallengeID, row.UserID, row.Email, row.Purpose, row.CodeHash, row.ExpiresAt, row.VerifiedAt, row.ConsumedAt, row.CreatedAt), nil
}

func (r *repository) MarkUserVerificationChallengeVerified(ctx context.Context, challengeID int32) error {
	if err := r.queries.MarkUserVerificationChallengeVerified(ctx, challengeID); err != nil {
		return fmt.Errorf("mark user verification challenge verified: %w", err)
	}

	return nil
}

func (r *repository) ConsumeUserVerificationChallenge(ctx context.Context, challengeID int32) error {
	if err := r.queries.ConsumeUserVerificationChallenge(ctx, challengeID); err != nil {
		return fmt.Errorf("consume user verification challenge: %w", err)
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

func toUserProfile(
	userID int32,
	userUUIDValue pgtype.UUID,
	email string,
	roleID int32,
	roleName string,
	firstName string,
	lastName string,
	phone string,
	profilePictureURL pgtype.Text,
	statusID int32,
	createdAt pgtype.Timestamptz,
) (UserProfile, error) {
	userUUID, err := fromPgUUID(userUUIDValue)
	if err != nil {
		return UserProfile{}, err
	}

	return UserProfile{
		UserID:            userID,
		UserUUID:          userUUID,
		RoleID:            roleID,
		RoleName:          roleName,
		FirstName:         firstName,
		LastName:          lastName,
		Email:             email,
		Phone:             phone,
		ProfilePictureURL: nullableString(profilePictureURL),
		StatusID:          statusID,
		CreatedAt:         createdAt.Time,
	}, nil
}

func toUserVerificationChallenge(
	challengeID int32,
	userID pgtype.Int4,
	email string,
	purpose string,
	codeHash string,
	expiresAt pgtype.Timestamptz,
	verifiedAt pgtype.Timestamptz,
	consumedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
) UserVerificationChallenge {
	return UserVerificationChallenge{
		ChallengeID: challengeID,
		UserID:      nullableInt32(userID),
		Email:       email,
		Purpose:     purpose,
		CodeHash:    codeHash,
		ExpiresAt:   expiresAt.Time,
		VerifiedAt:  nullableTime(verifiedAt),
		ConsumedAt:  nullableTime(consumedAt),
		CreatedAt:   createdAt.Time,
	}
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

func nullableInt32(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}

	result := value.Int32
	return &result
}

func nullableString(value pgtype.Text) *string {
	if !value.Valid || value.String == "" {
		return nil
	}

	result := value.String
	return &result
}
