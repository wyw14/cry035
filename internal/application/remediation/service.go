package remediation

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	baserepo "github.com/wyw14/cry035/internal/repository"
)

type Repository interface {
	GetPlan(context.Context, string) (maintenance.Plan, error)
	ListDefects(context.Context, string) ([]defect.Defect, error)
	SaveRectification(context.Context, defect.Rectification, int64, time.Time) (defect.Defect, error)
	SaveReinspection(context.Context, defect.Reinspection, time.Time) (maintenance.Plan, error)
	MarkReinspectionReviewed(context.Context, string, string, time.Time) (maintenance.Plan, error)
	RestoreEquipment(context.Context, string, int64, string, string, time.Time) (maintenance.Plan, error)
	AppendEvent(context.Context, audit.Event) error
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

func (s *Service) Defects(ctx context.Context, equipmentID string) ([]defect.Defect, error) {
	return s.repo.ListDefects(ctx, equipmentID)
}

type Rectify struct {
	DefectID        string
	ExpectedVersion int64
	Action          string
	Operator        string
	EvidenceID      string
	RequestID       string
}

func (s *Service) Rectify(ctx context.Context, input Rectify) (defect.Defect, error) {
	if input.Action == "" {
		return defect.Defect{}, fmt.Errorf("rectification action is required")
	}
	now := s.clock.Now().UTC()
	item := defect.Rectification{ID: s.ids.New(), DefectID: input.DefectID, Action: input.Action, Operator: input.Operator, EvidenceID: input.EvidenceID, CreatedAt: now}
	updated, err := s.repo.SaveRectification(ctx, item, input.ExpectedVersion, now)
	if err != nil {
		return defect.Defect{}, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{ID: s.ids.New(), EntityType: "equipment", EntityID: updated.EquipmentID, Action: "defect_rectified", Actor: input.Operator, RequestID: input.RequestID, OccurredAt: now, Details: map[string]any{"defect_id": updated.ID}})
	return updated, nil
}

type Reinspect struct {
	PlanID         string
	EquipmentID    string
	DefectIDs      []string
	RestrictionKey string
	Inspector      string
	Passed         bool
	Comment        string
	RequestID      string
}

func (s *Service) Reinspect(ctx context.Context, input Reinspect) (defect.Reinspection, maintenance.Plan, error) {
	if len(input.DefectIDs) == 0 {
		return defect.Reinspection{}, maintenance.Plan{}, fmt.Errorf("at least one defect is required")
	}
	now := s.clock.Now().UTC()
	item := defect.Reinspection{
		ID: s.ids.New(), PlanID: input.PlanID, EquipmentID: input.EquipmentID,
		DefectIDs: append([]string(nil), input.DefectIDs...), RestrictionKey: input.RestrictionKey,
		Inspector: input.Inspector, Passed: input.Passed, Comment: input.Comment, InspectedAt: now,
	}
	plan, err := s.repo.SaveReinspection(ctx, item, now)
	if err != nil {
		return defect.Reinspection{}, maintenance.Plan{}, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{ID: s.ids.New(), EntityType: "equipment", EntityID: input.EquipmentID, Action: "reinspection_recorded", Actor: input.Inspector, RequestID: input.RequestID, OccurredAt: now, Details: map[string]any{"reinspection_id": item.ID, "passed": item.Passed}})
	return item, plan, nil
}

func (s *Service) ReviewReinspection(ctx context.Context, id, reviewer, requestID string) (maintenance.Plan, error) {
	plan, err := s.repo.MarkReinspectionReviewed(ctx, id, reviewer, s.clock.Now())
	if err != nil {
		return maintenance.Plan{}, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{ID: s.ids.New(), EntityType: "equipment", EntityID: plan.EquipmentID, Action: "reinspection_qualified", Actor: reviewer, RequestID: requestID, OccurredAt: s.clock.Now(), Details: map[string]any{"plan_id": plan.ID}})
	return plan, nil
}

func (s *Service) Restore(ctx context.Context, planID string, expectedVersion int64, actor, requestID string) (maintenance.Plan, error) {
	plan, err := s.repo.GetPlan(ctx, planID)
	if err != nil {
		return maintenance.Plan{}, err
	}
	if plan.EverRestricted {
		defects, listErr := s.repo.ListDefects(ctx, plan.EquipmentID)
		if listErr != nil {
			return maintenance.Plan{}, listErr
		}
		if assessment := defect.AssessRestoration(plan.RestrictionKey, defects); !assessment.Allowed {
			return maintenance.Plan{}, fmt.Errorf("%w: %w", baserepo.ErrConflict, defect.ErrRestorationBlocked)
		}
	}
	return s.repo.RestoreEquipment(ctx, planID, expectedVersion, actor, requestID, s.clock.Now())
}
