package rentals

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

const dateLayout = "2006-01-02"

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/api/v1/rentals/preview", h.previewRental)
	r.POST("/api/v1/rentals", h.confirmRental)
}

// previewRental godoc
// @Summary      Preview rental pricing
// @Description  Calculates the rental price breakdown without confirming the rental. Only accessible to authenticated clients.
// @Tags         Rentals
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                 true  "Bearer access token"
// @Param        request        body      RentalPreviewRequest   true  "Rental preview payload"
// @Success      200            {object}  RentalPreviewResponse  "Rental price breakdown"
// @Failure      400            {object}  shared.ErrorResponse   "Invalid input"
// @Failure      403            {object}  shared.ErrorResponse   "Only clients can preview rentals"
// @Failure      404            {object}  shared.ErrorResponse   "Property not found"
// @Failure      422            {object}  shared.ErrorResponse   "Property is not rentable or the dates are blocked"
// @Failure      500            {object}  shared.ErrorResponse   "Internal error"
// @Router       /api/v1/rentals/preview [post]
func (h *Handler) previewRental(c *gin.Context) {
	auth, ok := resolveRentalAuth(c)
	if !ok {
		return
	}

	var req RentalPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	input, err := resolveRentalPreviewInput(req)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.PreviewRental(c.Request.Context(), auth, input)
	if err != nil {
		writeRentalError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// confirmRental godoc
// @Summary      Confirm rental
// @Description  Confirms the rental of an available property, commits the new transaction so contracts can read it, and then generates the digital contract through the internal POST /api/v1/contracts/rent endpoint. Only accessible to authenticated clients.
// @Tags         Rentals
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                true  "Bearer access token"
// @Param        request        body      RentalConfirmRequest  true  "Rental confirmation payload"
// @Success      201            {object}  RentalResponse        "Rental confirmed successfully"
// @Failure      400            {object}  shared.ErrorResponse  "Invalid input"
// @Failure      403            {object}  shared.ErrorResponse  "Only clients can confirm rentals"
// @Failure      404            {object}  shared.ErrorResponse  "Property not found"
// @Failure      422            {object}  shared.ErrorResponse  "Property is not rentable or the dates are blocked"
// @Failure      500            {object}  shared.ErrorResponse  "Internal error"
// @Router       /api/v1/rentals [post]
func (h *Handler) confirmRental(c *gin.Context) {
	auth, ok := resolveRentalAuth(c)
	if !ok {
		return
	}

	var req RentalConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	input, err := resolveRentalConfirmInput(req)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.ConfirmRental(c.Request.Context(), auth, input)
	if err != nil {
		writeRentalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func resolveRentalAuth(c *gin.Context) (AuthContext, bool) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return AuthContext{}, false
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return AuthContext{}, false
	}

	userUUIDValue, err := middleware.AuthenticatedUserUUID(c)
	if err != nil {
		shared.Unauthorized(c)
		return AuthContext{}, false
	}

	userUUID, err := uuid.Parse(userUUIDValue)
	if err != nil {
		shared.Unauthorized(c)
		return AuthContext{}, false
	}

	return AuthContext{
		UserID:     userID,
		RoleID:     roleID,
		UserUUID:   userUUID,
		AuthHeader: c.GetHeader("Authorization"),
	}, true
}

func resolveRentalPreviewInput(req RentalPreviewRequest) (RentalPreviewInput, error) {
	propertyUUID, err := resolveUUID(strings.TrimSpace(req.PropertyUUID), "property_uuid")
	if err != nil {
		return RentalPreviewInput{}, err
	}
	startDate, err := resolveDate(strings.TrimSpace(req.StartDate), "start_date")
	if err != nil {
		return RentalPreviewInput{}, err
	}
	endDate, err := resolveDate(strings.TrimSpace(req.EndDate), "end_date")
	if err != nil {
		return RentalPreviewInput{}, err
	}
	if req.PeriodID <= 0 {
		return RentalPreviewInput{}, errors.New("period_id must be greater than zero")
	}

	return RentalPreviewInput{
		PropertyUUID: propertyUUID,
		PeriodID:     req.PeriodID,
		StartDate:    startDate,
		EndDate:      endDate,
	}, nil
}

func resolveRentalConfirmInput(req RentalConfirmRequest) (RentalConfirmInput, error) {
	previewInput, err := resolveRentalPreviewInput(RentalPreviewRequest{
		PropertyUUID: req.PropertyUUID,
		PeriodID:     req.PeriodID,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
	})
	if err != nil {
		return RentalConfirmInput{}, err
	}

	clientUUID, err := resolveUUID(strings.TrimSpace(req.ClientUUID), "client_uuid")
	if err != nil {
		return RentalConfirmInput{}, err
	}

	return RentalConfirmInput{
		PropertyUUID: previewInput.PropertyUUID,
		ClientUUID:   clientUUID,
		PeriodID:     previewInput.PeriodID,
		StartDate:    previewInput.StartDate,
		EndDate:      previewInput.EndDate,
	}, nil
}

func resolveUUID(value, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, errors.New(field + " must be a valid UUID")
	}
	return parsed, nil
}

func resolveDate(value, field string) (time.Time, error) {
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, errors.New(field + " must use YYYY-MM-DD format")
	}
	return parsed.UTC(), nil
}

func writeRentalError(c *gin.Context, err error) {
	var statusErr *statusError
	if errors.As(err, &statusErr) {
		c.JSON(statusErr.StatusCode, gin.H{"error": statusErr.Message})
		return
	}

	shared.InternalError(c, "could not process rental request")
}

func formatDate(value time.Time) string {
	return value.UTC().Format(dateLayout)
}
