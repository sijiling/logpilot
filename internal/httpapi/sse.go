package httpapi

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/sijiling/logpilot/internal/diagnosis"
)

func (h *handler) streamDiagnosis(c *gin.Context) {
	request, ok := bindDiagnosis(c)
	if !ok || !h.acquire(c) {
		return
	}
	defer h.release()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	err := h.dependencies.Diagnoses.Stream(c.Request.Context(), request, func(event diagnosis.Event) error {
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	})
	if err != nil {
		data, _ := json.Marshal(map[string]string{"code": "UPSTREAM_ERROR", "message": "diagnosis failed"})
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\nevent: done\ndata: {}\n\n", data)
		c.Writer.Flush()
	}
}
