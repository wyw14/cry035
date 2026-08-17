package scheduler

import (
	"context"
	"time"

	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type Repository interface {
	ListPlans(context.Context) ([]maintenance.Plan, error)
	TransitionPlan(context.Context, string, int64, maintenance.Status, time.Time) (maintenance.Plan, error)
	SaveAlert(context.Context, audit.Alert) error
}

type IDGenerator interface{ New() string }
type Clock interface{ Now() time.Time }

type Service struct {
	planning *planning.Service
	repo     Repository
	ids      IDGenerator
	clock    Clock
}

func New(planningService *planning.Service, repo Repository, ids IDGenerator, clock Clock) *Service {
	return &Service{planning: planningService, repo: repo, ids: ids, clock: clock}
}

func (s *Service) RunOnce(ctx context.Context, horizon time.Time) ([]maintenance.Plan, error) {
	created, err := s.planning.GenerateDue(ctx, horizon, "待分配")
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	for _, plan := range plans {
		if (plan.Status == maintenance.StatusPlanned || plan.Status == maintenance.StatusSuspended) && plan.Window.End.Before(now) {
			updated, transitionErr := s.repo.TransitionPlan(ctx, plan.ID, plan.Version, maintenance.StatusOverdue, now)
			if transitionErr != nil {
				return nil, transitionErr
			}
			_ = s.repo.SaveAlert(ctx, audit.Alert{ID: s.ids.New(), EquipmentID: updated.EquipmentID, PlanID: updated.ID, Level: "warning", Message: "保养计划已逾期", CreatedAt: now})
		}
	}
	return created, nil
}
