package execution

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

var ErrUnknownEvidence = errors.New("measurement references unknown evidence")

type Repository interface {
	GetPlan(context.Context, string) (maintenance.Plan, error)
	GetProgram(context.Context, string) (maintenance.Program, error)
	SaveExecution(context.Context, inspection.Execution, []inspection.Finding, []defect.Defect, int64, time.Time) error
	GetExecutionByPlan(context.Context, string) (inspection.Execution, error)
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

type Submit struct {
	PlanID          string
	ExpectedVersion int64
	Technician      string
	Measurements    []inspection.Measurement
	Evidence        []inspection.Evidence
	RequestID       string
}

func (s *Service) Submit(ctx context.Context, input Submit) (inspection.Execution, []defect.Defect, error) {
	plan, err := s.repo.GetPlan(ctx, input.PlanID)
	if err != nil {
		return inspection.Execution{}, nil, err
	}
	if plan.Status != maintenance.StatusInProgress && plan.Status != maintenance.StatusOverdue {
		return inspection.Execution{}, nil, fmt.Errorf("plan status %s does not accept execution", plan.Status)
	}
	program, err := s.repo.GetProgram(ctx, plan.ProgramID)
	if err != nil {
		return inspection.Execution{}, nil, err
	}
	if err := inspection.ValidateSubmission(program.Checklist, input.Measurements, input.Evidence); err != nil {
		return inspection.Execution{}, nil, err
	}
	findings, err := inspection.Evaluate(program.Checklist, input.Measurements)
	if err != nil {
		return inspection.Execution{}, nil, err
	}
	now := s.clock.Now().UTC()
	execution := inspection.Execution{
		ID: s.ids.New(), PlanID: plan.ID, Technician: input.Technician,
		Checklist:    append([]maintenance.ChecklistItem(nil), program.Checklist...),
		Measurements: append([]inspection.Measurement(nil), input.Measurements...),
		Evidence:     append([]inspection.Evidence(nil), input.Evidence...), SubmittedAt: now,
	}
	defects := make([]defect.Defect, 0, len(findings))
	for _, finding := range findings {
		defects = append(defects, defect.Defect{
			ID: s.ids.New(), EquipmentID: plan.EquipmentID, PlanID: plan.ID, ExecutionID: execution.ID,
			ChecklistItem: finding.ItemID, Level: finding.Level, Description: finding.Reason,
			Status: defect.StatusOpen, DueAt: now.Add(7 * 24 * time.Hour), Version: 1,
			RestrictionKey: "restriction-" + plan.ID,
		})
	}
	if err := s.repo.SaveExecution(ctx, execution, findings, defects, input.ExpectedVersion, now); err != nil {
		return inspection.Execution{}, nil, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{
		ID: s.ids.New(), EntityType: "equipment", EntityID: plan.EquipmentID, Action: "execution_submitted",
		Actor: input.Technician, RequestID: input.RequestID, OccurredAt: now,
		Details: map[string]any{"plan_id": plan.ID, "execution_id": execution.ID, "finding_count": len(findings)},
	})
	return execution, defects, nil
}

func (s *Service) ByPlan(ctx context.Context, planID string) (inspection.Execution, error) {
	return s.repo.GetExecutionByPlan(ctx, planID)
}
