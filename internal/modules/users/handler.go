package users

import (
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
// @Summary      Registrar un nuevo usuario
// @Description  Crea un nuevo perfil de usuario en el sistema y envía confirmación a Supabase
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
// @Summary      Verificar cuenta de usuario
// @Description  Confirma el correo del usuario mediante un token enviado por email
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
// @Summary      Iniciar sesión
// @Description  Autentica al usuario y devuelve un token JWT
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
