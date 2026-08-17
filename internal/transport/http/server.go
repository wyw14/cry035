package httptransport

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/wyw14/cry035/internal/application/catalog"
	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/application/review"
	"github.com/wyw14/cry035/internal/middleware"
	"github.com/wyw14/cry035/internal/platform/blobstore"
	baserepo "github.com/wyw14/cry035/internal/repository"
	"github.com/wyw14/cry035/internal/service/alerting"
)

type Services struct {
	Catalog     *catalog.Service
	Planning    *planning.Service
	Execution   *execution.Service
	Review      *review.Service
	Remediation *remediation.Service
	Reporting   *reporting.Service
	Alerting    *alerting.Service
}

type Dependencies struct {
	Services        Services
	AttachmentStore *blobstore.Local
	RequestTimeout  time.Duration
	MaxUploadBytes  int64
	CORSOrigin      string
	Ready           func(context.Context) error
	Logger          *zap.Logger
}

type Server struct {
	services       Services
	attachments    *blobstore.Local
	maxUploadBytes int64
	ready          func(context.Context) error
	validator      *validator.Validate
	logger         *zap.Logger
}

func NewRouter(deps Dependencies) *gin.Engine {
	server := &Server{
		services: deps.Services, attachments: deps.AttachmentStore, maxUploadBytes: deps.MaxUploadBytes,
		ready: deps.Ready, validator: validator.New(), logger: deps.Logger,
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.RequestID(), middleware.Timeout(deps.RequestTimeout), middleware.SecurityHeaders(), middleware.CORS(deps.CORSOrigin), middleware.Identity())
	router.GET("/healthz", server.health)
	router.GET("/readyz", server.readiness)
	api := router.Group("/api/v1")
	server.registerCatalog(api)
	server.registerPlans(api)
	server.registerExecution(api)
	server.registerRemediation(api)
	server.registerReporting(api)
	return router
}

func (s *Server) health(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func (s *Server) readiness(c *gin.Context) {
	if s.ready != nil {
		if err := s.ready(c.Request.Context()); err != nil {
			s.error(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		s.error(c, err)
		return false
	}
	if err := s.validator.Struct(target); err != nil {
		s.error(c, err)
		return false
	}
	return true
}

func (s *Server) error(c *gin.Context, err error) {
	status := http.StatusBadRequest
	code := "VALIDATION_ERROR"
	switch {
	case errors.Is(err, baserepo.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, baserepo.ErrConflict):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, baserepo.ErrVersionConflict):
		status, code = http.StatusConflict, "VERSION_CONFLICT"
	case errors.Is(err, context.DeadlineExceeded):
		status, code = http.StatusGatewayTimeout, "REQUEST_TIMEOUT"
	}
	if s.logger != nil {
		s.logger.Warn("request failed", zap.String("request_id", middleware.GetRequestID(c)), zap.Error(err))
	}
	c.AbortWithStatusJSON(status, gin.H{"code": code, "message": err.Error(), "field_errors": []any{}, "request_id": middleware.GetRequestID(c)})
}

func page(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func respondPage[T any](c *gin.Context, items []T) {
	current, size := page(c)
	start := (current - 1) * size
	if start > len(items) {
		start = len(items)
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	c.JSON(http.StatusOK, gin.H{"items": items[start:end], "page": current, "page_size": size, "total": len(items), "request_id": middleware.GetRequestID(c)})
}
