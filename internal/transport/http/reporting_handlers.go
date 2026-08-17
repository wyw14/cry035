package httptransport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/middleware"
)

func (s *Server) registerReporting(api *gin.RouterGroup) {
	api.GET("/equipment/:id/history", s.equipmentHistory)
	api.GET("/equipment/:id/export.csv", s.exportEquipment)
	api.POST("/equipment/:id/vendor-services", middleware.RequireRole("planner", "safety_manager"), s.recordVendorService)
	api.GET("/alerts", s.alerts)
}

func (s *Server) equipmentHistory(c *gin.Context) {
	item, err := s.services.Reporting.History(c.Request.Context(), c.Param("id"))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) exportEquipment(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="equipment-history.csv"`)
	if err := s.services.Reporting.ExportCSV(c.Request.Context(), c.Param("id"), c.Writer); err != nil {
		// Headers may already be written; Gin still records the structured error for logs.
		s.error(c, err)
	}
}

type vendorServiceRequest struct {
	VendorName  string    `json:"vendor_name" validate:"required,max=128"`
	PlanID      string    `json:"plan_id"`
	Description string    `json:"description" validate:"required,max=1000"`
	AmountCents int64     `json:"amount_cents" validate:"gte=0"`
	Currency    string    `json:"currency" validate:"omitempty,len=3"`
	ServicedAt  time.Time `json:"serviced_at" validate:"required"`
}

func (s *Server) recordVendorService(c *gin.Context) {
	var request vendorServiceRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Reporting.RecordService(c.Request.Context(), reporting.RecordService{
		VendorName: request.VendorName, EquipmentID: c.Param("id"), PlanID: request.PlanID,
		Description: request.Description, AmountCents: request.AmountCents, Currency: request.Currency,
		ServicedAt: request.ServicedAt, Actor: middleware.Actor(c), RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) alerts(c *gin.Context) {
	items, err := s.services.Alerting.Alerts(c.Request.Context())
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}
