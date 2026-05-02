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
		var errRes map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errRes)
		return "", fmt.Errorf("supabase error %d: %v", resp.StatusCode, errRes["msg"])
	}

	var res struct {
		ID   string `json:"id"`
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("error decodificando JSON: %w", err)
	}

	finalID := res.User.ID
	if finalID == "" {
		finalID = res.ID
	}

	if finalID == "" {
		return "", fmt.Errorf("no se encontró el UUID en la respuesta de Supabase")
	}

	return finalID, nil
}

func (s *service) VerifyUser(ctx context.Context, email, token string) error {
	url := fmt.Sprintf("%s/auth/v1/verify", s.config.SupabaseURL)

	body, _ := json.Marshal(map[string]string{
		"email": email,
		"token": token,
		"type":  "signup",
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.config.SupabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+s.config.SupabaseAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("código inválido o expirado (status: %d)", resp.StatusCode)
	}

	user, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("usuario verificado en auth pero no encontrado en db: %w", err)
	}

	err = s.repository.UpdateUserStatus(ctx, user.UserID, 2)
	if err != nil {
		return fmt.Errorf("error al activar usuario en db: %w", err)
	}

	return nil
}

func (s *service) LoginUser(ctx context.Context, input LoginInput) (LoginResult, error) {
	url := fmt.Sprintf("%s/auth/v1/token?grant_type=password", s.config.SupabaseURL)

	body, _ := json.Marshal(map[string]string{
		"email":    input.Email,
		"password": input.Password,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return LoginResult{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.config.SupabaseAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return LoginResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LoginResult{}, fmt.Errorf("credenciales inválidas o cuenta no verificada")
	}

	var authRes struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authRes); err != nil {
		return LoginResult{}, fmt.Errorf("error decodificando tokens: %w", err)
	}

	userData, err := s.repository.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return LoginResult{}, fmt.Errorf("error obteniendo datos del usuario: %w", err)
	}

	return LoginResult{
		AccessToken:  authRes.AccessToken,
		RefreshToken: authRes.RefreshToken,
		User:         userData,
	}, nil
}
