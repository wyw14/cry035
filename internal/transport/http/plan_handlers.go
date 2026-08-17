package httptransport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/middleware"
)

func (s *Server) registerPlans(api *gin.RouterGroup) {
	api.GET("/plans", s.listPlans)
	api.POST("/plans", middleware.RequireRole("safety_manager", "planner"), s.createPlan)
	api.POST("/plans/generate", middleware.RequireRole("safety_manager", "planner"), s.generatePlans)
	api.POST("/plans/:id/start", middleware.RequireRole("safety_manager", "planner", "technician"), s.startPlan)
	api.POST("/plans/:id/suspend", middleware.RequireRole("safety_manager", "planner", "technician"), s.suspendPlan)
	api.POST("/plans/:id/cancel", middleware.RequireRole("safety_manager", "planner"), s.cancelPlan)
}

func (s *Server) listPlans(c *gin.Context) {
	items, err := s.services.Planning.List(c.Request.Context())
	if err != nil {
		s.error(c, err)
		return
	}
	respondPage(c, items)
}

type createPlanRequest struct {
	EquipmentID string                         `json:"equipment_id" validate:"required"`
	ProgramID   string                         `json:"program_id" validate:"required"`
	Start       time.Time                      `json:"start" validate:"required"`
	End         time.Time                      `json:"end" validate:"required"`
	Assignee    string                         `json:"assignee" validate:"required"`
	Spares      []maintenance.SpareRequirement `json:"spares"`
}

func (s *Server) createPlan(c *gin.Context) {
	var request createPlanRequest
	if !s.bind(c, &request) {
		return
	}
	key := c.GetHeader("Idempotency-Key")
	item, created, err := s.services.Planning.Create(c.Request.Context(), planning.CreatePlan{
		EquipmentID: request.EquipmentID, ProgramID: request.ProgramID, Start: request.Start, End: request.End,
		Assignee: request.Assignee, Spares: request.Spares, IdempotencyKey: key,
		Actor: middleware.Actor(c), RequestID: middleware.GetRequestID(c),
	})
	if err != nil {
		s.error(c, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"item": item, "created": created, "request_id": middleware.GetRequestID(c)})
}

type generatePlansRequest struct {
	Horizon time.Time `json:"horizon" validate:"required"`
}

func (s *Server) generatePlans(c *gin.Context) {
	var request generatePlansRequest
	if !s.bind(c, &request) {
		return
	}
	items, err := s.services.Planning.GenerateDue(c.Request.Context(), request.Horizon, "待分配")
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "created": len(items), "request_id": middleware.GetRequestID(c)})
}

type transitionRequest struct {
	ExpectedVersion int64 `json:"expected_version" validate:"required,min=1"`
}

func (s *Server) startPlan(c *gin.Context)   { s.transitionPlan(c, maintenance.StatusInProgress) }
func (s *Server) suspendPlan(c *gin.Context) { s.transitionPlan(c, maintenance.StatusSuspended) }
func (s *Server) cancelPlan(c *gin.Context)  { s.transitionPlan(c, maintenance.StatusCancelled) }

func (s *Server) transitionPlan(c *gin.Context, target maintenance.Status) {
	var request transitionRequest
	if !s.bind(c, &request) {
		return
	}
	item, err := s.services.Planning.Transition(c.Request.Context(), c.Param("id"), request.ExpectedVersion, target, middleware.Actor(c), middleware.GetRequestID(c))
	if err != nil {
		s.error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item, "request_id": middleware.GetRequestID(c)})
}
