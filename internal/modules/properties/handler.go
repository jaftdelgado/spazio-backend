package properties

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service PropertyService
}

func NewHandler(service PropertyService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/properties", h.createProperty)
}

func (h *Handler) createProperty(c *gin.Context) {
	var req CreatePropertyInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}
	if err := validateCreatePropertyRequest(req); err != nil {
		shared.BadRequest(c, err)
		return
	}
	result, err := h.service.CreateProperty(c.Request.Context(), CreatePropertyInput{
		OwnerID:        req.OwnerID,
		Title:          strings.TrimSpace(req.Title),
		Description:    strings.TrimSpace(req.Description),
		PropertyTypeID: req.PropertyTypeID,
		ModalityID:     req.ModalityID,
		StatusID:       req.StatusID,
		CoverPhotoURL:  strings.TrimSpace(req.CoverPhotoURL),
	})
	if err != nil {
		shared.InternalError(c, "could not create property")
		return
	}
	c.JSON(http.StatusCreated, result)
}

func validateCreatePropertyRequest(req CreatePropertyInput) error {
	return shared.Validate([]shared.ValidationRule{
		{Fail: req.OwnerID <= 0, Msg: "owner_id is required"},
		{Fail: strings.TrimSpace(req.Title) == "", Msg: "title is required"},
		{Fail: strings.TrimSpace(req.Description) == "", Msg: "description is required"},
		{Fail: req.PropertyTypeID <= 0, Msg: "property_type_id is required"},
		{Fail: req.ModalityID <= 0, Msg: "modality_id is required"},
		{Fail: req.StatusID <= 0, Msg: "status_id is required"},
		{Fail: strings.TrimSpace(req.CoverPhotoURL) == "", Msg: "cover_photo_url is required"},
	})
}
