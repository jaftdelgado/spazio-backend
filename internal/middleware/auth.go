package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Auth(supabaseURL, supabaseAnonKey string, db *pgxpool.Pool) gin.HandlerFunc {
	client := &http.Client{Timeout: 10 * time.Second}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No se proporcionó token de acceso"})
			c.Abort()
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Formato de token inválido"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		userUUID, err := validateSupabaseToken(c.Request.Context(), client, supabaseURL, supabaseAnonKey, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido o expirado"})
			c.Abort()
			return
		}

		c.Set("user_uuid", userUUID)

		var roleName string
		query := `
			SELECT r.name 
			FROM users u 
			JOIN roles r ON u.role_id = r.id 
			WHERE u.user_uuid = $1`

		err = db.QueryRow(c.Request.Context(), query, userUUID).Scan(&roleName)
		if err != nil {
			roleName = "user"
		}

		c.Set("user_role", roleName)

		fmt.Println("Pase por el Middleware de Auth")
		c.Next()
	}
}

func validateSupabaseToken(ctx context.Context, client *http.Client, supabaseURL, supabaseAnonKey, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(supabaseURL, "/")+"/auth/v1/user", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("supabase auth returned status %d", resp.StatusCode)
	}

	var user struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", err
	}

	if user.ID == "" {
		return "", fmt.Errorf("supabase user response did not include id")
	}

	return user.ID, nil
}
