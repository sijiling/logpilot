package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sijiling/logpilot/internal/session"
)

func (h *handler) getSession(c *gin.Context) {
	record, messages, err := h.dependencies.Sessions.Find(c.Request.Context(), c.Param("sessionID"))
	if errors.Is(err, session.ErrNotFound) {
		writeError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "session lookup failed")
		return
	}
	type messageResponse struct {
		Role      session.Role `json:"role"`
		Content   string       `json:"content"`
		CreatedAt any          `json:"created_at"`
	}
	responseMessages := make([]messageResponse, len(messages))
	for index, message := range messages {
		responseMessages[index] = messageResponse{Role: message.Role, Content: message.Content, CreatedAt: message.CreatedAt}
	}
	c.JSON(http.StatusOK, gin.H{"id": record.ID, "created_at": record.CreatedAt, "updated_at": record.UpdatedAt, "messages": responseMessages})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetHeader("X-Request-ID")}})
}
