package users

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
)

type Handler struct {
	service   UserService
	cookieCfg CookieConfig
}

type CookieConfig struct {
	Secure             bool
	AccessTokenMaxAge  int
	RefreshTokenMaxAge int
}

func NewHandler(service UserService, cookieCfg CookieConfig) *Handler {
	return &Handler{
		service:   service,
		cookieCfg: cookieCfg,
	}
}

// PreRegister godoc
// @Summary      Start email verification
// @Description  Sends a six digit verification code to an email before creating a user account.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        input  body      PreRegisterInput  true  "Email to verify"
// @Success      200    {object}  MessageResult
// @Failure      400    {object}  map[string]string "Invalid request body"
// @Failure      409    {object}  map[string]string "Email already registered"
// @Failure      500    {object}  map[string]string "Internal server error"
// @Router       /api/v1/users/pre-register [post]
func (h *Handler) PreRegister(c *gin.Context) {
	var input PreRegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email inválido"})
		return
	}
	input.Email = normalizeEmail(input.Email)

	if err := h.service.PreRegisterUser(c.Request.Context(), input); err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, MessageResult{Message: "Código de verificación enviado."})
}

// VerifyEmail godoc
// @Summary      Verify email code
// @Description  Validates the email verification code and returns a temporary verification token.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        input  body      VerifyEmailInput  true  "Email and verification code"
// @Success      200    {object}  VerifyEmailResult
// @Failure      400    {object}  map[string]string "Invalid request body"
// @Failure      401    {object}  map[string]string "Invalid code"
// @Failure      404    {object}  map[string]string "Verification not found"
// @Failure      410    {object}  map[string]string "Code expired"
// @Router       /api/v1/users/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var input VerifyEmailInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de verificación inválidos"})
		return
	}
	input.Email = normalizeEmail(input.Email)
	input.Code = strings.TrimSpace(input.Code)

	result, err := h.service.VerifyEmail(c.Request.Context(), input)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Register godoc
// @Summary      Complete user registration
// @Description  Creates an active local user after email verification using a temporary verification token.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      CompleteRegisterInput  true  "Verified registration data"
// @Success      201   {object}  RegisterResult
// @Failure      400   {object}  map[string]string "Invalid request body"
// @Failure      401   {object}  map[string]string "Invalid verification token"
// @Failure      409   {object}  map[string]string "Email already taken"
// @Failure      500   {object}  map[string]string "Internal server error"
// @Router       /api/v1/users/register [post]
func (h *Handler) Register(c *gin.Context) {
	var input CompleteRegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de datos inválido"})
		return
	}
	input = sanitizeCompleteRegisterInput(input)
	if input.FirstName == "" || input.LastName == "" || len(input.Password) < minPasswordLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de registro inválidos"})
		return
	}

	result, err := h.service.CompleteRegister(c.Request.Context(), input)
	if err != nil {
		writeUserError(c, err)
		return
	}

	loginResult, err := h.service.LoginUser(c.Request.Context(), LoginInput{
		Email:    result.User.Email,
		Password: input.Password,
	})
	if err != nil {
		writeUserError(c, err)
		return
	}
	setAuthCookies(c, loginResult.AccessToken, loginResult.RefreshToken, h.cookieCfg)

	c.JSON(http.StatusCreated, result)
}

// Login godoc
// @Summary      Login
// @Description  Authenticates an active local user and returns an access token plus an opaque refresh token.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        login  body      LoginInput  true  "Login credentials"
// @Success      200    {object}  LoginResult
// @Failure      400    {object}  map[string]string "Invalid request body"
// @Failure      401    {object}  map[string]string "Invalid credentials"
// @Router       /api/v1/users/login [post]
func (h *Handler) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email o contraseña requeridos"})
		return
	}
	input.Email = normalizeEmail(input.Email)

	result, err := h.service.LoginUser(c.Request.Context(), input)
	if err != nil {
		writeUserError(c, err)
		return
	}
	setAuthCookies(c, result.AccessToken, result.RefreshToken, h.cookieCfg)

	c.JSON(http.StatusOK, result)
}

// Refresh godoc
// @Summary      Refresh session
// @Description  Rotates a refresh token and returns a new access token plus a new refresh token.
// @Tags         Users
// @Produce      json
// @Param        Cookie  header    string  true  "spazio_refresh_token=<token>"
// @Success      200      {object}  RefreshResult
// @Failure      401      {object}  map[string]string "Invalid refresh token"
// @Router       /api/v1/users/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("spazio_refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token requerido"})
		return
	}

	result, err := h.service.RefreshToken(c.Request.Context(), RefreshInput{RefreshToken: refreshToken})
	if err != nil {
		writeUserError(c, err)
		return
	}
	setAuthCookies(c, result.AccessToken, result.RefreshToken, h.cookieCfg)

	c.JSON(http.StatusOK, result)
}

// Logout godoc
// @Summary      Logout
// @Description  Revokes the provided refresh token for the authenticated user.
// @Tags         Users
// @Produce      json
// @Param        Authorization  header    string        true  "Bearer access token"
// @Param        Cookie         header    string  true  "spazio_refresh_token=<token>"
// @Success      200            {object}  MessageResult
// @Failure      401            {object}  map[string]string "Invalid session or refresh token"
// @Router       /api/v1/users/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("spazio_refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token requerido"})
		return
	}

	if err := h.service.LogoutUser(c.Request.Context(), RefreshInput{RefreshToken: refreshToken}); err != nil {
		writeUserError(c, err)
		return
	}
	clearAuthCookies(c, h.cookieCfg)

	c.JSON(http.StatusOK, MessageResult{Message: "Sesión cerrada correctamente."})
}

// UpdateProfile godoc
// GetProfile godoc
// @Summary      Get authenticated user profile
// @Description  Returns the profile data of the currently authenticated user.
// @Tags         Users
// @Produce      json
// @Param        Authorization  header    string   true  "Bearer access token"
// @Success      200            {object}  AuthUser
// @Failure      401            {object}  map[string]string "Invalid session"
// @Failure      404            {object}  map[string]string "User not found"
// @Router       /api/v1/users/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	userUUID, err := middleware.AuthenticatedUserUUID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida o expirada"})
		return
	}

	user, err := h.service.GetProfile(c.Request.Context(), userUUID)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates the authenticated user's local profile data.
// @Tags         Users
// @Produce      json
// @Param        Authorization  header    string              true  "Bearer access token"
// @Param        profile        body      UpdateProfileInput  true  "Profile data"
// @Success      200            {object}  UpdateProfileResult
// @Failure      400            {object}  map[string]string "Invalid request body"
// @Failure      401            {object}  map[string]string "Invalid session"
// @Failure      404            {object}  map[string]string "User not found"
// @Router       /api/v1/users/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userUUID, err := middleware.AuthenticatedUserUUID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida o expirada"})
		return
	}

	var input UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos"})
		return
	}
	input = sanitizeUpdateProfileInput(input)
	if input.FirstName == "" || input.LastName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nombre y apellido son obligatorios"})
		return
	}

	result, err := h.service.UpdateProfile(c.Request.Context(), userUUID, input)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteAccount godoc
// @Summary      Delete user account
// @Description  Soft deletes the authenticated user's local account and revokes active refresh tokens.
// @Tags         Users
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer access token"
// @Success      200            {object}  MessageResult
// @Failure      401            {object}  map[string]string "Invalid session"
// @Failure      404            {object}  map[string]string "User not found"
// @Router       /api/v1/users/profile [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	userUUID, err := middleware.AuthenticatedUserUUID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida o expirada"})
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), userUUID); err != nil {
		writeUserError(c, err)
		return
	}

	clearAuthCookies(c, h.cookieCfg)
	c.JSON(http.StatusOK, MessageResult{Message: "Cuenta eliminada correctamente."})
}

func sanitizeCompleteRegisterInput(input CompleteRegisterInput) CompleteRegisterInput {
	input.VerificationToken = strings.TrimSpace(input.VerificationToken)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.ProfilePictureURL = strings.TrimSpace(input.ProfilePictureURL)
	return input
}

func setAuthCookies(c *gin.Context, accessToken, refreshToken string, cfg CookieConfig) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("spazio_access_token", accessToken, cfg.AccessTokenMaxAge, "/", "", cfg.Secure, true)
	c.SetCookie("spazio_refresh_token", refreshToken, cfg.RefreshTokenMaxAge, "/", "", cfg.Secure, true)
}

func clearAuthCookies(c *gin.Context, cfg CookieConfig) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("spazio_access_token", "", -1, "/", "", cfg.Secure, true)
	c.SetCookie("spazio_refresh_token", "", -1, "/", "", cfg.Secure, true)
}

func sanitizeUpdateProfileInput(input UpdateProfileInput) UpdateProfileInput {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.ProfilePictureURL = strings.TrimSpace(input.ProfilePictureURL)
	return input
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func writeUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
	case errors.Is(err, ErrVerificationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Verificación no encontrada"})
	case errors.Is(err, ErrEmailTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "El email ya está registrado"})
	case errors.Is(err, ErrEmailAlreadyVerified):
		c.JSON(http.StatusConflict, gin.H{"error": "El email ya fue verificado"})
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
	case errors.Is(err, ErrInvalidVerificationToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token de verificación inválido"})
	case errors.Is(err, ErrCodeExpired):
		c.JSON(http.StatusGone, gin.H{"error": "El código expiró"})
	case errors.Is(err, ErrCodeInvalid):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Código inválido"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno del servidor"})
	}
}
