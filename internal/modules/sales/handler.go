package sales

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/api/v1/sales", h.confirmSale)
}

// confirmSale godoc
// @Summary      Formalize property sale
// @Description  Formalizes the sale of an available property. Only accessible to authenticated users with the Agent role.
// @Tags         Sales
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string        true  "Bearer access token"
// @Param        request        body      SaleRequest   true  "Sale confirmation payload"
// @Success      201            {object}  SaleResponse  "Sale formalized successfully"
// @Failure      400            {object}  shared.ErrorResponse  "Invalid request body"
// @Failure      403            {object}  shared.ErrorResponse  "Only agents can formalize sales"
// @Failure      404            {object}  shared.ErrorResponse  "Property not found"
// @Failure      422            {object}  shared.ErrorResponse  "Property is not saleable or amount does not match the current sale price"
// @Failure      500            {object}  shared.ErrorResponse  "Internal error or contract generation failure"
// @Router       /api/v1/sales [post]
func (h *Handler) confirmSale(c *gin.Context) {
	auth, ok := resolveSaleAuth(c)
	if !ok {
		return
	}

	var req SaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	input, err := resolveSaleInput(req)
	if err != nil {
		shared.BadRequest(c, err)
		return
	}

	result, err := h.service.ConfirmSale(c.Request.Context(), auth, input)
	if err != nil {
		writeSaleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func resolveSaleAuth(c *gin.Context) (AuthContext, bool) {
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

	return AuthContext{
		UserID:     userID,
		RoleID:     roleID,
		AuthHeader: c.GetHeader("Authorization"),
	}, true
}

func resolveSaleInput(req SaleRequest) (SaleInput, error) {
	propertyUUID, err := resolveUUID(strings.TrimSpace(req.PropertyUUID), "property_uuid")
	if err != nil {
		return SaleInput{}, err
	}

	if req.AgreedAmount <= 0 {
		return SaleInput{}, errors.New("agreed_amount must be greater than zero")
	}

	return SaleInput{
		PropertyUUID: propertyUUID,
		AgreedAmount: req.AgreedAmount,
	}, nil
}

func resolveUUID(value, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, errors.New(field + " must be a valid UUID")
	}

	return parsed, nil
}

func writeSaleError(c *gin.Context, err error) {
	var statusErr *statusError
	if errors.As(err, &statusErr) {
		c.JSON(statusErr.StatusCode, gin.H{"error": statusErr.Message})
		return
	}

	shared.InternalError(c, "could not process sale request")
}
