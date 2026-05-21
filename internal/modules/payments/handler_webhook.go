package payments

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// @Summary Handle MercadoPago Webhooks
// @Description Webhook receiver for asynchronous payment updates.
// @Tags Payments
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/payments/webhook [post]
func (h *Handler) handleWebhook(c *gin.Context) {
	xSignature := c.GetHeader("x-signature")
	xRequestID := c.GetHeader("x-request-id")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read body"})
		return
	}

	err = h.service.HandleWebhook(c.Request.Context(), xSignature, xRequestID, body)
	if err != nil {
		// F3: Improved operational visibility for failed webhooks
		errMsg := err.Error()
		if strings.Contains(errMsg, "signature") || strings.Contains(errMsg, "timestamp") {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "invalid security signature"})
			return
		}
		// For other errors, we might still want to return 200 to MercadoPago to stop retries
		// but log it internally. For now, returning 400 to distinguish from success.
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": errMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}
