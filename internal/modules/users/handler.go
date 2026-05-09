package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service UserService
}

func NewHandler(service UserService) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary      Register new User
// @Description  Create new User
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      CreateUserInput  true  "Datos del nuevo usuario"
// @Success      201   {object}  map[string]interface{} "Usuario creado exitosamente"
// @Failure      400   {object}  map[string]string      "Formato de datos inválido"
// @Failure      500   {object}  map[string]string      "Error interno del servidor"
// @Router       /users/register [post]
func (h *Handler) Register(c *gin.Context) {
	var input CreateUserInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Formato de datos inválido",
			"details": err.Error(),
		})
		return
	}

	result, err := h.service.RegisterUser(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "No se pudo completar el registro",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// Verify godoc
// @Summary      Verify user account
// @Description  Verifies the user's email address using the token sent to their email.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        verify  body      VerifyUserInput  true  "Token y email de verificación"
// @Success      200     {object}  map[string]string      "Cuenta verificada exitosamente"
// @Failure      400     {object}  map[string]string      "Datos de verificación inválidos"
// @Failure      401     {object}  map[string]string      "Verificación fallida"
// @Router       /users/verify [post]
func (h *Handler) Verify(c *gin.Context) {
	var input VerifyUserInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Datos de verificación inválidos",
			"details": err.Error(),
		})
		return
	}

	err := h.service.VerifyUser(c.Request.Context(), input.Email, input.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Verificación fallida",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "¡Cuenta verificada exitosamente!",
	})
}

// Login godoc
// @Summary      Login
// @Description  Authenticates the user and returns a JWT token.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        login  body      LoginInput  true  "Credenciales de acceso"
// @Success      200    {object}  map[string]interface{} "Login exitoso"
// @Failure      400    {object}  map[string]string      "Email o contraseña requeridos"
// @Failure      401    {object}  map[string]string      "Login fallido"
// @Router       /users/login [post]
func (h *Handler) Login(c *gin.Context) {
	var input LoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email o contraseña requeridos"})
		return
	}

	result, err := h.service.LoginUser(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Login fallido",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates the authenticated user's profile data using the Supabase access token to resolve the user identity.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string           true  "Bearer access token"
// @Param        profile        body      UpdateUserInput  true  "Profile data to update"
// @Success      200            {object}  map[string]interface{} "Profile updated successfully"
// @Failure      400            {object}  map[string]string      "Invalid request body"
// @Failure      401            {object}  map[string]string      "Invalid or expired session"
// @Failure      500            {object}  map[string]string      "Profile update failed"
// @Router       /users/profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	userUUID, ok := authenticatedUserUUID(c)
	if !ok {
		return
	}

	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de entrada inválidos: " + err.Error()})
		return
	}

	result, err := h.service.UpdateUser(c.Request.Context(), userUUID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo actualizar el perfil: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Perfil actualizado correctamente",
		"data":    result,
	})
}

// DeleteAccount godoc
// @Summary      Delete user account
// @Description  Soft deletes the authenticated user's local account by marking deleted_at. The Supabase account may still exist, but deleted local users cannot log in through this API.
// @Tags         Users
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer access token"
// @Success      200            {object}  map[string]string "Account deleted successfully"
// @Failure      401            {object}  map[string]string "Invalid or expired session"
// @Failure      404            {object}  map[string]string "User not found"
// @Failure      500            {object}  map[string]string "Account deletion failed"
// @Router       /users/DeleteProfile [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	userUUID, ok := authenticatedUserUUID(c)
	if !ok {
		return
	}

	userEmail, _ := authenticatedUserEmail(c)

	if err := h.service.DeleteUser(c.Request.Context(), userUUID, userEmail); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo eliminar la cuenta: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cuenta eliminada correctamente"})
}

func authenticatedUserUUID(c *gin.Context) (string, bool) {
	userUUIDValue, exists := c.Get("user_uuid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida o expirada"})
		return "", false
	}

	userUUID, ok := userUUIDValue.(string)
	if !ok || userUUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identificador de usuario inválido"})
		return "", false
	}

	return userUUID, true
}

func authenticatedUserEmail(c *gin.Context) (string, bool) {
	userEmailValue, exists := c.Get("user_email")
	if !exists {
		return "", false
	}

	userEmail, ok := userEmailValue.(string)
	if !ok || userEmail == "" {
		return "", false
	}

	return userEmail, true
}
