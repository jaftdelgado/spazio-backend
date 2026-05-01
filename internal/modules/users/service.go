package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaftdelgado/spazio-backend/internal/config"
)

type service struct {
	repository UserRepository
	config     *config.Config
}

func NewService(repository UserRepository, cfg *config.Config) UserService {
	return &service{
		repository: repository,
		config:     cfg,
	}
}

func (s *service) RegisterUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error) {
	supabaseUUID, err := s.signUpWithSupabase(input.Email, input.PasswordHash)
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("supabase auth error: %w", err)
	}

	input.UserUUID = supabaseUUID

	result, err := s.repository.CreateUser(ctx, input)
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("neon database error: %w", err)
	}

	return result, nil
}

func (s *service) signUpWithSupabase(email, password string) (string, error) {
	url := fmt.Sprintf("%s/auth/v1/signup", s.config.SupabaseURL)

	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	fmt.Println("URL Final:", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.config.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+s.config.SupabaseAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("supabase returned status: %d", resp.StatusCode)
	}

	var res struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.User.ID, nil
}
