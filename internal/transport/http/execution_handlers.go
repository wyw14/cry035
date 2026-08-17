package httptransport

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/review"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/middleware"
)

func (s *Server) registerExecution(api *gin.RouterGroup) {
	api.POST("/plans/:id/execution", middleware.RequireRole("technician", "safety_manager"), s.submitExecution)
	api.GET("/plans/:id/execution", s.getExecution)
	api.POST("/plans/:id/review", middleware.RequireRole("reviewer", "safety_manager"), s.reviewExecution)
	api.POST("/attachments", middleware.RequireRole("technician", "reviewer", "safety_manager"), s.uploadAttachment)
}

type submitExecutionRequest struct {
	ExpectedVersion int64                    `json:"expected_version" validate:"required,min=1"`
	Measurements    []inspection.Measurement `json:"measurements" validate:"required"`
	Evidence        []inspection.Evidence    `json:"evidence"`
}

func (s *Server) submitExecution(c *gin.Context) {
	var request submitExecutionRequest
	if !s.bind(c, &request) {
		return
	}
	item, defects, err := s.services.Execution.Submit(c.Request.Context(), execution.Submit{
		PlanID: c.Param("id"), ExpectedVersion: request.ExpectedVersion, Technician: middleware.Actor(c),
		Measurements: request.Measurements, Evidence: request.Evidence, RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "defects": defects, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) getExecution(c *gin.Context) {
	item, err := s.services.Execution.ByPlan(c.Request.Context(), c.Param("id"))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

type reviewExecutionRequest struct {
	ExpectedVersion int64                    `json:"expected_version" validate:"required,min=1"`
	Outcome         inspection.ReviewOutcome `json:"outcome" validate:"required,oneof=passed failed"`
	Comment         string                   `json:"comment" validate:"required,max=1000"`
}

func (s *Server) reviewExecution(c *gin.Context) {
	var request reviewExecutionRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Review.Decide(c.Request.Context(), review.Decide{
		PlanID: c.Param("id"), ExpectedVersion: request.ExpectedVersion, Reviewer: middleware.Actor(c),
		Outcome: request.Outcome, Comment: request.Comment, RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) uploadAttachment(c *gin.Context) {
	if s.attachments == nil {
		s.error(c, http.ErrNotSupported)
		return
	}
	if s.maxUploadBytes > 0 {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, s.maxUploadBytes)
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		s.error(c, err)
		return
	}
	defer file.Close()
	id := middleware.GetRequestID(c)
	contentType := header.Header.Get("Content-Type")
	content, err := io.ReadAll(io.LimitReader(file, s.maxUploadBytes+1))
	if err != nil {
		s.error(c, err)
		return
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(content)
	}
	saved, err := s.attachments.Save(c.Request.Context(), id, contentType, bytes.NewReader(content))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": inspection.Evidence{
		ID: id, FileName: header.Filename, ContentType: contentType, Size: saved.Size, SHA256: saved.SHA256, StoredPath: saved.Path, CreatedAt: time.Now().UTC(),
	}, "request_id": id})
}
