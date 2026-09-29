package httpapi

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *handler) uploadDocument(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	file, err := c.FormFile("file")
	if err != nil || strings.ToLower(filepath.Ext(file.Filename)) != ".md" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "a Markdown file is required")
		return
	}
	opened, err := file.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "cannot open file")
		return
	}
	defer opened.Close()
	content, err := io.ReadAll(io.LimitReader(opened, (2<<20)+1))
	if err != nil || len(content) > 2<<20 {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "document exceeds 2 MiB")
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		title = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	}
	result, err := h.dependencies.Documents.Index(c.Request.Context(), title, content)
	if err != nil {
		writeError(c, http.StatusBadGateway, "UPSTREAM_ERROR", "document indexing failed")
		return
	}
	status := http.StatusCreated
	if result.Duplicated {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"document_id": result.Document.ID, "title": result.Document.Title, "status": result.Document.Status, "chunk_count": result.Document.ChunkCount, "duplicated": result.Duplicated})
}
