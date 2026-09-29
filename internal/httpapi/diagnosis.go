package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sijiling/logpilot/internal/diagnosis"
)

func (h *handler) diagnose(c *gin.Context) {
	request, ok := bindDiagnosis(c)
	if !ok || !h.acquire(c) {
		return
	}
	defer h.release()
	response, err := h.dependencies.Diagnoses.Diagnose(c.Request.Context(), request)
	if err != nil {
		writeError(c, http.StatusBadGateway, "UPSTREAM_ERROR", "diagnosis failed")
		return
	}
	c.JSON(http.StatusOK, response)
}

func bindDiagnosis(c *gin.Context) (diagnosis.Request, bool) {
	var request diagnosis.Request
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 70<<10)
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Question) == "" || len([]rune(request.Question)) > 4000 || len(request.Logs) > 64<<10 {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "question is required and request limits must be respected")
		return diagnosis.Request{}, false
	}
	return request, true
}

func (h *handler) acquire(c *gin.Context) bool {
	select {
	case h.diagnoses <- struct{}{}:
		return true
	default:
		writeError(c, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "diagnosis capacity is full")
		return false
	}
}
func (h *handler) release() { <-h.diagnoses }
