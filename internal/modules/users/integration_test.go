//go:build integration

package users

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
)

func TestIntegration_UsersRepository_UserLifecycleAndChallenges(t *testing.T) {
	pool := shared.SetupTestDB(t)
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	shared.WithTransaction(t, pool, func(tx pgx.Tx) {
		repo := &repository{queries: sqlcgen.New(tx)}

		email := fmt.Sprintf("integration-user-%d@example.com", suffix)
		userUUID := uuid.New().String()

		created, err := repo.CreateUser(ctx, CreateUserRecord{
			UserUUID:     userUUID,
			RoleID:       3,
			FirstName:    "Integration",
			LastName:     "User",
			Email:        email,
			PasswordHash: "hashed-password",
			Phone:        "5550001111",
			StatusID:     1,
		})
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}
		if created.UserID == 0 || created.UserUUID != userUUID {
			t.Fatalf("unexpected created user: %+v", created)
		}

		byEmail, err := repo.GetUserByEmail(ctx, email)
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v", err)
		}
		if byEmail.UserID != created.UserID {
			t.Fatalf("user id mismatch: got %d want %d", byEmail.UserID, created.UserID)
		}

		profile, err := repo.UpdateProfile(ctx, userUUID, UpdateProfileInput{
			FirstName: "Updated",
			LastName:  "Name",
			Phone:     "5559990000",
		})
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if profile.FirstName != "Updated" || profile.LastName != "Name" || profile.Phone != "5559990000" {
			t.Fatalf("unexpected updated profile: %+v", profile)
		}

		updatedEmail := fmt.Sprintf("integration-user-updated-%d@example.com", suffix)
		profile, err = repo.UpdateUserEmail(ctx, created.UserID, updatedEmail)
		if err != nil {
			t.Fatalf("UpdateUserEmail() error = %v", err)
		}
		if profile.Email != updatedEmail {
			t.Fatalf("email mismatch: got %q want %q", profile.Email, updatedEmail)
		}

		if err := repo.UpdateUserPassword(ctx, created.UserID, "new-hash"); err != nil {
			t.Fatalf("UpdateUserPassword() error = %v", err)
		}

		pendingID, err := repo.CreatePendingVerification(ctx, updatedEmail, "code-hash", time.Now().Add(time.Hour))
		if err != nil {
			t.Fatalf("CreatePendingVerification() error = %v", err)
		}
		pending, err := repo.GetLatestPendingVerification(ctx, updatedEmail)
		if err != nil {
			t.Fatalf("GetLatestPendingVerification() error = %v", err)
		}
		if pending.VerificationID != pendingID {
			t.Fatalf("verification id mismatch: got %d want %d", pending.VerificationID, pendingID)
		}
		if err := repo.MarkPendingVerificationVerified(ctx, pendingID); err != nil {
			t.Fatalf("MarkPendingVerificationVerified() error = %v", err)
		}

		challengeID, err := repo.CreateUserVerificationChallenge(ctx, CreateChallengeRecord{
			UserID:    &created.UserID,
			Email:     updatedEmail,
			Purpose:   challengeResetPwd,
			CodeHash:  "challenge-hash",
			ExpiresAt: time.Now().Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("CreateUserVerificationChallenge() error = %v", err)
		}

		challenge, err := repo.GetLatestUserVerificationChallenge(ctx, updatedEmail, challengeResetPwd)
		if err != nil {
			t.Fatalf("GetLatestUserVerificationChallenge() error = %v", err)
		}
		if challenge.ChallengeID != challengeID {
			t.Fatalf("challenge id mismatch: got %d want %d", challenge.ChallengeID, challengeID)
		}

		if err := repo.MarkUserVerificationChallengeVerified(ctx, challengeID); err != nil {
			t.Fatalf("MarkUserVerificationChallengeVerified() error = %v", err)
		}
		if err := repo.ConsumeUserVerificationChallenge(ctx, challengeID); err != nil {
			t.Fatalf("ConsumeUserVerificationChallenge() error = %v", err)
		}

		if err := repo.CreateRefreshToken(ctx, created.UserID, "refresh-hash", time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("CreateRefreshToken() error = %v", err)
		}
		token, err := repo.GetRefreshToken(ctx, "refresh-hash")
		if err != nil {
			t.Fatalf("GetRefreshToken() error = %v", err)
		}
		if token.UserID != created.UserID {
			t.Fatalf("refresh token user mismatch: got %d want %d", token.UserID, created.UserID)
		}
		if err := repo.RevokeRefreshToken(ctx, "refresh-hash"); err != nil {
			t.Fatalf("RevokeRefreshToken() error = %v", err)
		}
	})
}
