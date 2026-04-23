// Package handlers exposes HTTP handlers for the API.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/services"
)

type propertyCreator interface {
	CreateProperty(ctx context.Context, input services.CreatePropertyInput) (services.CreatePropertyResult, error)
}

// Handler exposes property HTTP endpoints.
type Handler struct {
	creator propertyCreator
}

// NewPropertyHandler builds a property handler.
func NewPropertyHandler(creator propertyCreator) *Handler {
	return &Handler{creator: creator}
}

type createPropertyRequest struct {
	OwnerID        int32  `json:"owner_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PropertyTypeID int32  `json:"property_type_id"`
	ModalityID     int32  `json:"modality_id"`
	StatusID       int32  `json:"status_id"`
	CoverPhotoURL  string `json:"cover_photo_url"`
}

// CreateProperty handles POST /properties.
func (h *Handler) CreateProperty(c *gin.Context) {
	var req createPropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := validateCreatePropertyRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.creator.CreateProperty(c.Request.Context(), services.CreatePropertyInput{
		OwnerID:        req.OwnerID,
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		PropertyTypeID: req.PropertyTypeID,
		ModalityID:     req.ModalityID,
		StatusID:       req.StatusID,
		CoverPhotoURL:  strings.TrimSpace(req.CoverPhotoURL),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create property"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func validateCreatePropertyRequest(req createPropertyRequest) error {
	if req.OwnerID <= 0 {
		return fmt.Errorf("owner_id is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if req.PropertyTypeID <= 0 {
		return fmt.Errorf("property_type_id is required")
	}
	if req.ModalityID <= 0 {
		return fmt.Errorf("modality_id is required")
	}
	if req.StatusID <= 0 {
		return fmt.Errorf("status_id is required")
	}
	if strings.TrimSpace(req.CoverPhotoURL) == "" {
		return fmt.Errorf("cover_photo_url is required")
	}

	return nil
}
