package uploads

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service UploadsService
}

func NewHandler(service UploadsService) *Handler {
	return &Handler{
		service: service,
	}
}

// uploadPropertyPhoto godoc
// @Summary Upload property photo
// @Description Uploads a property photo to object storage and registers metadata. Max size is 5MB and allowed types are image/jpeg, image/png, and image/webp.
// @Tags        Uploads
// @Accept      multipart/form-data
// @Produce     json
// @Param       property_uuid  path      string  true   "Property UUID"
// @Param       file           formData  file    true   "Image file"
// @Param       label          formData  string  false  "Optional label"
// @Param       alt_text       formData  string  false  "Optional alt text"
// @Param       sort_order     formData  int     false  "Sort order" default(0)
// @Param       is_cover       formData  bool    false  "Mark as cover" default(false)
// @Success     201            {object}  UploadPhotoResult  "Photo uploaded"
// @Failure     400            {object}  shared.ErrorResponse "Invalid input"
// @Failure     404            {object}  shared.ErrorResponse "Property not found"
// @Failure     500            {object}  shared.ErrorResponse "Internal error"
// @Router      /api/v1/uploads/properties/{property_uuid}/photos [post]
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
		if errors.Is(err, ErrPropertyNotFound) {
			shared.NotFound(c, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not upload property photo"})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *Handler) uploadPropertyPhotos(c *gin.Context) {
	propertyUUID := strings.TrimSpace(c.Param("property_uuid"))
	if propertyUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property_uuid is required"})
		return
	}

	if _, err := uuid.Parse(propertyUUID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property_uuid must be a valid UUID"})
		return
	}

	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	headers := c.Request.MultipartForm.File["file"]
	if len(headers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one file is required"})
		return
	}
	if len(headers) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 10 files allowed"})
		return
	}

	for i, header := range headers {
		if err := validateUploadPhotoHeader(header); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file[%d]: %s", i, err.Error())})
			return
		}
	}

	photos := make([]UploadPhotoInput, 0, len(headers))
	defer closeUploadFiles(photos)

	for i, header := range headers {
		file, err := header.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file[%d]: could not open file", i)})
			return
		}

		photos = append(photos, UploadPhotoInput{
			PropertyUUID: propertyUUID,
			MimeType:     header.Header.Get("Content-Type"),
			File:         file,
		})
	}

	ctx := c.Request.Context()
	result, err := h.service.UploadPropertyPhotos(ctx, UploadPhotosInput{
		PropertyUUID: propertyUUID,
		Photos:       photos,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, result)
		return
	}

	if len(result.Uploaded) > 0 && len(result.Failed) > 0 {
		c.JSON(http.StatusMultiStatus, result)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func validateUploadPhotoHeader(header *multipart.FileHeader) error {
	if header.Size > 5*1024*1024 {
		return errors.New("file size must be less than 5MB")
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
		return errors.New("allowed MIME types: image/jpeg, image/png, image/webp")
	}

	return nil
}

func closeUploadFiles(photos []UploadPhotoInput) {
	for _, photo := range photos {
		closer, ok := photo.File.(interface{ Close() error })
		if !ok {
			continue
		}
		if err := closer.Close(); err != nil {
			continue
		}
	}
}
