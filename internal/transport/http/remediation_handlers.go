package httptransport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/middleware"
)

func (s *Server) registerRemediation(api *gin.RouterGroup) {
	api.GET("/defects", s.listDefects)
	api.POST("/defects/:id/rectify", middleware.RequireRole("technician", "safety_manager"), s.rectifyDefect)
	api.POST("/plans/:id/reinspection", middleware.RequireRole("reviewer", "safety_manager"), s.reinspect)
	api.POST("/reinspections/:id/review", middleware.RequireRole("reviewer", "safety_manager"), s.reviewReinspection)
	api.POST("/plans/:id/restore", middleware.RequireRole("safety_manager"), s.restorePlan)
}

func (s *Server) listDefects(c *gin.Context) {
	items, err := s.services.Remediation.Defects(c.Request.Context(), c.Query("equipment_id"))
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}

type rectifyRequest struct {
	ExpectedVersion int64  `json:"expected_version" validate:"required,min=1"`
	Action          string `json:"action" validate:"required,max=2000"`
	EvidenceID      string `json:"evidence_id"`
}

func (s *Server) rectifyDefect(c *gin.Context) {
	var request rectifyRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Remediation.Rectify(c.Request.Context(), remediation.Rectify{
		DefectID: c.Param("id"), ExpectedVersion: request.ExpectedVersion, Action: request.Action,
		Operator: middleware.Actor(c), EvidenceID: request.EvidenceID, RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

type reinspectRequest struct {
	EquipmentID    string   `json:"equipment_id" validate:"required"`
	DefectIDs      []string `json:"defect_ids" validate:"required,min=1"`
	RestrictionKey string   `json:"restriction_key" validate:"required"`
	Passed         bool     `json:"passed"`
	Comment        string   `json:"comment" validate:"max=1000"`
}

func (s *Server) reinspect(c *gin.Context) {
	var request reinspectRequest
	if !s.bind(c, &request) {
		return
	}
	item, plan, err := s.services.Remediation.Reinspect(c.Request.Context(), remediation.Reinspect{
		PlanID: c.Param("id"), EquipmentID: request.EquipmentID, DefectIDs: request.DefectIDs,
		RestrictionKey: request.RestrictionKey, Inspector: middleware.Actor(c), Passed: request.Passed,
		Comment: request.Comment, RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": item, "plan": plan, "request_id": middleware.GetRequestID(c)})
}

func (s *Server) reviewReinspection(c *gin.Context) {
	item, err := s.services.Remediation.ReviewReinspection(c.Request.Context(), c.Param("id"), middleware.Actor(c), middleware.GetRequestID(c))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}

type restoreRequest struct {
	ExpectedVersion int64 `json:"expected_version" validate:"required,min=1"`
}

func (s *Server) restorePlan(c *gin.Context) {
	var request restoreRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Remediation.Restore(c.Request.Context(), c.Param("id"), request.ExpectedVersion, middleware.Actor(c), middleware.GetRequestID(c))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}
