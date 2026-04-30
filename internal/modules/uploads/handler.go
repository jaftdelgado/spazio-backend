package uploads

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service UploadsService
}

func NewHandler(service UploadsService) *Handler {
	return &Handler{
		service: service,
	}
}

// @Summary Upload property photo
// @Description Uploads a photo for a property to R2 and registers it
// @Tags uploads
// @Accept mpfd
// @Produce json
// @Param property_uuid path string true "Property UUID"
// @Param file formData file true "The image file"
// @Param label formData string false "Label"
// @Param alt_text formData string false "Alt Text"
// @Param sort_order formData int false "Sort Order" default(0)
// @Param is_cover formData bool false "Is Cover" default(false)
// @Success 201 {object} UploadPhotoResult
// @Router /uploads/properties/{property_uuid}/photos [post]
func (h *Handler) uploadPropertyPhoto(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("property_uuid"))
	if propertyUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property_uuid is required"})
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property_uuid must be a valid UUID"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be less than 5MB"})
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed MIME types: image/jpeg, image/png, image/webp"})
		return
	}

	var label *string
	if l := strings.TrimSpace(c.PostForm("label")); l != "" {
		label = &l
	}

	var altText *string
	if a := strings.TrimSpace(c.PostForm("alt_text")); a != "" {
		altText = &a
	}

	sortOrder := int32(0)
	if s := strings.TrimSpace(c.PostForm("sort_order")); s != "" {
		val, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			sortOrder = int32(val)
		}
	}

	isCover := false
	if isC := strings.TrimSpace(c.PostForm("is_cover")); isC != "" {
		val, err := strconv.ParseBool(isC)
		if err == nil {
			isCover = val
		}
	}

	ctx := c.Request.Context()

	result, err := h.service.UploadPropertyPhoto(ctx, UploadPhotoInput{
		PropertyUUID: propertyUUID,
		MimeType:     mimeType,
		Label:        label,
		AltText:      altText,
		SortOrder:    sortOrder,
		IsCover:      isCover,
		File:         file,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not upload property photo"})
		return
	}

	c.JSON(http.StatusCreated, result)
}
