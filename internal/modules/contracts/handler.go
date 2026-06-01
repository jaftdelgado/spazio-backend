package contracts

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jaftdelgado/spazio-backend/internal/middleware"
	"github.com/jaftdelgado/spazio-backend/internal/shared"
)

type Handler struct {
	service ContractService
}

func NewHandler(service ContractService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/api/v1/contracts/rent", h.createRentContract)
	r.POST("/api/v1/contracts/sale", h.createSaleContract)
	r.GET("/api/v1/contracts", h.listContracts)
	r.GET("/api/v1/contracts/:uuid", h.getContract)
}

// listContracts godoc
// @Summary      List contracts
// @Description  Returns a paginated list of contracts. Admins and Agents see all, Owners see only their own. The authenticated user is resolved from the bearer token.
// @Tags         Contracts
// @Produce      json
// @Param        Authorization     header    string  true   "Bearer access token"
// @Param        page              query     int     false  "Page number (default 1)"
// @Param        limit             query     int     false  "Items per page (default 10)"
// @Param        transaction_type  query     string  false  "Filter by type (sale/rent)"
// @Param        status_id         query     int     false  "Filter by status ID"
// @Param        owner_id          query     int     false  "Filter by owner ID (Admin/Agent only)"
// @Param        start_date        query     string  false  "Filter by start date (RFC3339)"
// @Param        end_date          query     string  false  "Filter by end date (RFC3339)"
// @Param        search            query     string  false  "Search by property title or client name"
// @Success      200               {array}   ContractListItem
// @Failure      400               {object}  shared.ErrorResponse
// @Failure      401               {object}  shared.ErrorResponse
// @Failure      500               {object}  shared.ErrorResponse
// @Router       /api/v1/contracts [get]
func (h *Handler) listContracts(c *gin.Context) {
	userID, roleID, ok := resolveAuthenticatedContractIdentity(c)
	if !ok {
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		shared.BadRequest(c, fmt.Errorf("invalid page number: %s", pageStr))
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		shared.BadRequest(c, fmt.Errorf("invalid limit: %s", limitStr))
		return
	}

	filter := ListContractsFilter{
		Page:  int32(page),
		Limit: int32(limit),
	}

	if tType := c.Query("transaction_type"); tType != "" {
		if tType != "rent" && tType != "sale" {
			shared.BadRequest(c, fmt.Errorf("invalid transaction_type: %s", tType))
			return
		}

		filter.TransactionType = &tType
	}

	if sID := c.Query("status_id"); sID != "" {
		val, err := strconv.Atoi(sID)
		if err != nil || val <= 0 {
			shared.BadRequest(c, fmt.Errorf("invalid status_id: %s", sID))
			return
		}

		v32 := int32(val)
		filter.StatusID = &v32
	}

	if oID := c.Query("owner_id"); oID != "" {
		val, err := strconv.Atoi(oID)
		if err != nil || val <= 0 {
			shared.BadRequest(c, fmt.Errorf("invalid owner_id: %s", oID))
			return
		}

		v32 := int32(val)
		filter.OwnerID = &v32
	}

	if sDate := c.Query("start_date"); sDate != "" {
		t, err := time.Parse(time.RFC3339, sDate)
		if err != nil {
			shared.BadRequest(c, fmt.Errorf("invalid start_date: %s", sDate))
			return
		}

		filter.StartDate = &t
	}

	if eDate := c.Query("end_date"); eDate != "" {
		t, err := time.Parse(time.RFC3339, eDate)
		if err != nil {
			shared.BadRequest(c, fmt.Errorf("invalid end_date: %s", eDate))
			return
		}

		filter.EndDate = &t
	}

	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	result, err := h.service.ListContracts(c.Request.Context(), userID, roleID, filter)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// getContract godoc
// @Summary      Get contract detail
// @Description  Returns the full details of a contract, including the PDF URL, using the authenticated session.
// @Tags         Contracts
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer access token"
// @Param        uuid           path      string  true  "Contract UUID"
// @Success      200            {object}  ContractDetail
// @Failure      400            {object}  shared.ErrorResponse
// @Failure      401            {object}  shared.ErrorResponse
// @Failure      403            {object}  shared.ErrorResponse
// @Failure      404            {object}  shared.ErrorResponse
// @Failure      500            {object}  shared.ErrorResponse
// @Router       /api/v1/contracts/{uuid} [get]
func (h *Handler) getContract(c *gin.Context) {
	userID, roleID, ok := resolveAuthenticatedContractIdentity(c)
	if !ok {
		return
	}

	contractUUID, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		shared.BadRequest(c, errors.New("invalid uuid format"))
		return
	}

	result, err := h.service.GetContractDetail(c.Request.Context(), userID, roleID, contractUUID)
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "no tiene permiso") {
			shared.Forbidden(c, errMsg)
			return
		}

		if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no encontrado") {
			shared.NotFound(c, errMsg)
			return
		}

		shared.InternalError(c, errMsg)
		return
	}

	c.JSON(http.StatusOK, result)
}

// createRentContract godoc
// @Summary      Generate digital rent contract
// @Description  Generates a legal rent contract in PDF format based on real estate transaction data and stores it in R2. Only the client can invoke this endpoint.
// @Tags         Contracts
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                   true  "Bearer access token"
// @Param        request        body      CreateRentContractInput  true  "Rent contract generation data"
// @Success      201            {object}  CreateContractResult     "Contract generated and stored successfully"
// @Failure      400            {object}  shared.ErrorResponse     "Invalid input or logical date error"
// @Failure      401            {object}  shared.ErrorResponse     "Missing or invalid authentication"
// @Failure      403            {object}  shared.ErrorResponse     "Unauthorized user (not the client)"
// @Failure      500            {object}  shared.ErrorResponse     "Internal error in PDF generation or storage"
// @Router       /api/v1/contracts/rent [post]
func (h *Handler) createRentContract(c *gin.Context) {
	userID, _, ok := resolveAuthenticatedContractIdentity(c)
	if !ok {
		return
	}

	var req CreateRentContractInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if req.TransactionID <= 0 {
		shared.BadRequest(c, errors.New("transaction_id is required"))
		return
	}

	result, err := h.service.GenerateRentContract(c.Request.Context(), userID, req)
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "no tiene permiso") || strings.Contains(errMsg, "no autorizada") {
			shared.Forbidden(c, errMsg)
			return
		}

		if strings.Contains(errMsg, "ya existe") ||
			strings.Contains(errMsg, "no coincide") ||
			strings.Contains(errMsg, "posterior") ||
			strings.Contains(errMsg, "corresponde") {
			shared.BadRequest(c, err)
			return
		}

		shared.InternalError(c, errMsg)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// createSaleContract godoc
// @Summary      Generate digital sale contract
// @Description  Generates a legal sale contract in PDF format based on real estate transaction data and stores it in R2. Only the assigned property agent can invoke this endpoint.
// @Tags         Contracts
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string                   true  "Bearer access token"
// @Param        request        body      CreateSaleContractInput  true  "Sale contract generation data"
// @Success      201            {object}  CreateContractResult     "Contract generated and stored successfully"
// @Failure      400            {object}  shared.ErrorResponse     "Invalid input or logical error"
// @Failure      401            {object}  shared.ErrorResponse     "Missing or invalid authentication"
// @Failure      403            {object}  shared.ErrorResponse     "Unauthorized user (not the assigned property agent)"
// @Failure      500            {object}  shared.ErrorResponse     "Internal error in PDF generation or storage"
// @Router       /api/v1/contracts/sale [post]
func (h *Handler) createSaleContract(c *gin.Context) {
	userID, _, ok := resolveAuthenticatedContractIdentity(c)
	if !ok {
		return
	}

	var req CreateSaleContractInput
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, err)
		return
	}

	if req.TransactionID <= 0 {
		shared.BadRequest(c, errors.New("transaction_id is required"))
		return
	}

	result, err := h.service.GenerateSaleContract(c.Request.Context(), userID, req)
	if err != nil {
		errMsg := err.Error()

		if strings.Contains(errMsg, "no tiene permiso") || strings.Contains(errMsg, "no autorizada") {
			shared.Forbidden(c, errMsg)
			return
		}

		if strings.Contains(errMsg, "ya existe") ||
			strings.Contains(errMsg, "no coincide") ||
			strings.Contains(errMsg, "corresponde") {
			shared.BadRequest(c, err)
			return
		}

		shared.InternalError(c, errMsg)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func resolveAuthenticatedContractIdentity(c *gin.Context) (int32, int32, bool) {
	userID, err := middleware.AuthenticatedUserID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	roleID, err := middleware.AuthenticatedRoleID(c)
	if err != nil {
		shared.Unauthorized(c)
		return 0, 0, false
	}

	return userID, roleID, true
}
