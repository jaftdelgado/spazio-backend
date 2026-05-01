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
