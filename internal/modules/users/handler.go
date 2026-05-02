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
