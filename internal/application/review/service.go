package review

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type Repository interface {
	GetPlan(context.Context, string) (maintenance.Plan, error)
	GetExecutionByPlan(context.Context, string) (inspection.Execution, error)
	ApplyReview(context.Context, string, string, string, string, inspection.ReviewOutcome, int64, string, time.Time) (maintenance.Plan, error)
	AppendEvent(context.Context, audit.Event) error
	SaveAlert(context.Context, audit.Alert) error
}

type IDGenerator interface{ New() string }
type Clock interface{ Now() time.Time }

type Service struct {
	repo  Repository
	ids   IDGenerator
	clock Clock
}

func New(repo Repository, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, ids: ids, clock: clock}
}

type Decide struct {
	PlanID          string
	ExpectedVersion int64
	Reviewer        string
	Outcome         inspection.ReviewOutcome
	Comment         string
	RequestID       string
}

func (s *Service) Decide(ctx context.Context, input Decide) (maintenance.Plan, error) {
	if input.Outcome != inspection.ReviewPassed && input.Outcome != inspection.ReviewFailed {
		return maintenance.Plan{}, fmt.Errorf("invalid review outcome")
	}
	plan, err := s.repo.GetPlan(ctx, input.PlanID)
	if err != nil {
		return maintenance.Plan{}, err
	}
	execution, err := s.repo.GetExecutionByPlan(ctx, input.PlanID)
	if err != nil {
		return maintenance.Plan{}, err
	}
	now := s.clock.Now().UTC()
	restrictionKey := "restriction-" + plan.ID
	updated, err := s.repo.ApplyReview(ctx, plan.ID, execution.ID, input.Reviewer, input.Comment, input.Outcome, input.ExpectedVersion, restrictionKey, now)
	if err != nil {
		return maintenance.Plan{}, err
	}
	action := "maintenance_qualified"
	if input.Outcome == inspection.ReviewFailed {
		action = "equipment_restricted"
		_ = s.repo.SaveAlert(ctx, audit.Alert{
			ID: s.ids.New(), EquipmentID: plan.EquipmentID, PlanID: plan.ID, Level: "critical",
			Message: "复核未通过，设备已限用并等待整改复检", CreatedAt: now,
		})
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{
		ID: s.ids.New(), EntityType: "equipment", EntityID: plan.EquipmentID, Action: action,
		Actor: input.Reviewer, RequestID: input.RequestID, OccurredAt: now,
		Details: map[string]any{"plan_id": plan.ID, "outcome": input.Outcome},
	})
	return updated, nil
}
