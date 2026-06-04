package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService defines the operations required to issue and validate access tokens.
type JWTService interface {
	Generate(userID int32, userUUID string, roleID int32, roleName string) (string, error)
	Validate(tokenString string) (*Claims, error)
}

// Claims contains the authenticated user data embedded in an access token.
type Claims struct {
	UserID   int32  `json:"user_id"`
	UserUUID string `json:"user_uuid"`
	RoleID   int32  `json:"role_id"`
	RoleName string `json:"role_name"`
	jwt.RegisteredClaims
}

type jwtService struct {
	secret        []byte
	expiryMinutes int
}

// NewJWTService creates a JWT service that signs HS256 access tokens.
func NewJWTService(secret string, expiryMinutes int) JWTService {
	if expiryMinutes <= 0 {
		expiryMinutes = 60
	}

	return &jwtService{
		secret:        []byte(secret),
		expiryMinutes: expiryMinutes,
	}
}

func (s *jwtService) Generate(userID int32, userUUID string, roleID int32, roleName string) (string, error) {
	if len(s.secret) == 0 {
		return "", errors.New("jwt secret is not configured")
	}

	now := time.Now().UTC()
	claims := Claims{
		UserID:   userID,
		UserUUID: userUUID,
		RoleID:   roleID,
		RoleName: roleName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	return tokenString, nil
}

func (s *jwtService) Validate(tokenString string) (*Claims, error) {
	if len(s.secret) == 0 {
		return nil, errors.New("jwt secret is not configured")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected jwt signing method: %s", token.Header["alg"])
		}

		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("validate jwt: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid jwt")
	}

	return claims, nil
}
