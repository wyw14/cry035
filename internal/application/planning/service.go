package planning

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type Repository interface {
	GetEquipment(context.Context, string) (equipment.Equipment, error)
	GetModel(context.Context, string) (equipment.Model, error)
	ListEquipment(context.Context) ([]equipment.Equipment, error)
	GetProgram(context.Context, string) (maintenance.Program, error)
	ListPrograms(context.Context) ([]maintenance.Program, error)
	CreatePlan(context.Context, maintenance.Plan) (maintenance.Plan, bool, error)
	GetPlan(context.Context, string) (maintenance.Plan, error)
	ListPlans(context.Context) ([]maintenance.Plan, error)
	TransitionPlan(context.Context, string, int64, maintenance.Status, time.Time) (maintenance.Plan, error)
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

type CreatePlan struct {
	EquipmentID    string
	ProgramID      string
	Start          time.Time
	End            time.Time
	Assignee       string
	Spares         []maintenance.SpareRequirement
	IdempotencyKey string
	Actor          string
	RequestID      string
}

func (s *Service) Create(ctx context.Context, input CreatePlan) (maintenance.Plan, bool, error) {
	if _, err := s.repo.GetEquipment(ctx, input.EquipmentID); err != nil {
		return maintenance.Plan{}, false, err
	}
	program, err := s.repo.GetProgram(ctx, input.ProgramID)
	if err != nil {
		return maintenance.Plan{}, false, err
	}
	window, err := maintenance.NewWindow(input.Start, input.End)
	if err != nil {
		return maintenance.Plan{}, false, err
	}
	now := s.clock.Now().UTC()
	plan := maintenance.Plan{
		ID: s.ids.New(), EquipmentID: input.EquipmentID, ProgramID: input.ProgramID,
		ProgramVersion: program.Version, GenerationKey: input.IdempotencyKey, Window: window,
		Assignee: input.Assignee, Spares: append([]maintenance.SpareRequirement(nil), input.Spares...),
		Status: maintenance.StatusPlanned, Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	stored, created, err := s.repo.CreatePlan(ctx, plan)
	if err != nil || !created {
		return stored, created, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{
		ID: s.ids.New(), EntityType: "equipment", EntityID: input.EquipmentID, Action: "plan_created",
		Actor: input.Actor, RequestID: input.RequestID, OccurredAt: now,
		Details: map[string]any{"plan_id": plan.ID, "window_start": window.Start, "window_end": window.End},
	})
	return stored, true, nil
}

func (s *Service) List(ctx context.Context) ([]maintenance.Plan, error) {
	return s.repo.ListPlans(ctx)
}

func (s *Service) Transition(ctx context.Context, planID string, expectedVersion int64, to maintenance.Status, actor, requestID string) (maintenance.Plan, error) {
	plan, err := s.repo.TransitionPlan(ctx, planID, expectedVersion, to, s.clock.Now())
	if err != nil {
		return maintenance.Plan{}, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{
		ID: s.ids.New(), EntityType: "equipment", EntityID: plan.EquipmentID, Action: "plan_status_changed",
		Actor: actor, RequestID: requestID, OccurredAt: s.clock.Now(),
		Details: map[string]any{"plan_id": plan.ID, "status": plan.Status},
	})
	return plan, nil
}

func (s *Service) GenerateDue(ctx context.Context, horizon time.Time, assignee string) ([]maintenance.Plan, error) {
	equipmentItems, err := s.repo.ListEquipment(ctx)
	if err != nil {
		return nil, err
	}
	programs, err := s.repo.ListPrograms(ctx)
	if err != nil {
		return nil, err
	}
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	calendar := newGenerationCalendar(plans)
	created := make([]maintenance.Plan, 0)
	now := s.clock.Now().UTC()
	for _, equipmentItem := range equipmentItems {
		model, modelErr := s.repo.GetModel(ctx, equipmentItem.ModelID)
		if modelErr != nil {
			return nil, modelErr
		}
		for _, program := range programs {
			if !program.Active || !program.AppliesTo(model.Category) {
				continue
			}
			anchor := equipmentItem.CreatedAt
			for _, plan := range plans {
				if plan.EquipmentID == equipmentItem.ID && plan.ProgramID == program.ID && (plan.Status == maintenance.StatusQualified || plan.Status == maintenance.StatusRestored) && plan.Window.End.After(anchor) {
					anchor = plan.Window.End
				}
			}
			due := program.NextDue(anchor).UTC()
			for !due.After(now) {
				anchor = due
				due = program.NextDue(anchor).UTC()
			}
			if due.After(horizon.UTC()) {
				continue
			}
			window := calendar.Next(equipmentItem.ID, due)
			key := fmt.Sprintf("%s/%s/v%d/%s", equipmentItem.ID, program.ID, program.Version, due.Format("2006-01-02"))
			plan, wasCreated, createErr := s.Create(ctx, CreatePlan{
				EquipmentID: equipmentItem.ID, ProgramID: program.ID, Start: window.Start, End: window.End,
				Assignee: assignee, IdempotencyKey: key, Actor: "local-scheduler", RequestID: key,
			})
			if createErr != nil {
				return nil, createErr
			}
			if wasCreated {
				calendar.Reserve(plan)
				created = append(created, plan)
			}
		}
	}
	sort.Slice(created, func(i, j int) bool { return created[i].Window.Start.Before(created[j].Window.Start) })
	return created, nil
}
