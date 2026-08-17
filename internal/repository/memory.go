package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/domain/supplier"
)

type MemoryStore struct {
	mu              sync.RWMutex
	buildings       map[string]equipment.Building
	models          map[string]equipment.Model
	equipment       map[string]equipment.Equipment
	programs        map[string]maintenance.Program
	plans           map[string]maintenance.Plan
	executions      map[string]inspection.Execution
	executionByPlan map[string]string
	defects         map[string]defect.Defect
	rectifications  map[string]defect.Rectification
	reinspections   map[string]defect.Reinspection
	services        map[string]supplier.ServiceRecord
	events          []audit.Event
	alerts          map[string]audit.Alert
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		buildings:       make(map[string]equipment.Building),
		models:          make(map[string]equipment.Model),
		equipment:       make(map[string]equipment.Equipment),
		programs:        make(map[string]maintenance.Program),
		plans:           make(map[string]maintenance.Plan),
		executions:      make(map[string]inspection.Execution),
		executionByPlan: make(map[string]string),
		defects:         make(map[string]defect.Defect),
		rectifications:  make(map[string]defect.Rectification),
		reinspections:   make(map[string]defect.Reinspection),
		services:        make(map[string]supplier.ServiceRecord),
		alerts:          make(map[string]audit.Alert),
	}
}

func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *MemoryStore) Seed(buildings []equipment.Building, models []equipment.Model, equipmentItems []equipment.Equipment, programs []maintenance.Program) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, item := range buildings {
		if _, exists := s.buildings[item.ID]; !exists {
			s.buildings[item.ID] = item
		}
	}
	for _, item := range models {
		if _, exists := s.models[item.ID]; !exists {
			s.models[item.ID] = item
		}
	}
	for _, item := range equipmentItems {
		if _, exists := s.equipment[item.ID]; !exists {
			s.equipment[item.ID] = item
		}
	}
	for _, item := range programs {
		if _, exists := s.programs[item.ID]; !exists {
			s.programs[item.ID] = item
		}
	}
}

func (s *MemoryStore) ListBuildings(ctx context.Context) ([]equipment.Building, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]equipment.Building, 0, len(s.buildings))
	for _, item := range s.buildings {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *MemoryStore) ListEquipment(ctx context.Context) ([]equipment.Equipment, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]equipment.Equipment, 0, len(s.equipment))
	for _, item := range s.equipment {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	return items, nil
}

func (s *MemoryStore) GetModel(ctx context.Context, id string) (equipment.Model, error) {
	if err := checkContext(ctx); err != nil {
		return equipment.Model{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.models[id]
	if !ok {
		return equipment.Model{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) GetEquipment(ctx context.Context, id string) (equipment.Equipment, error) {
	if err := checkContext(ctx); err != nil {
		return equipment.Equipment{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.equipment[id]
	if !ok {
		return equipment.Equipment{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) SaveEquipment(ctx context.Context, item equipment.Equipment) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if err := item.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.equipment {
		if existing.Code == item.Code && existing.ID != item.ID {
			return ErrConflict
		}
	}
	s.equipment[item.ID] = item
	return nil
}

func (s *MemoryStore) ListPrograms(ctx context.Context) ([]maintenance.Program, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]maintenance.Program, 0, len(s.programs))
	for _, item := range s.programs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *MemoryStore) GetProgram(ctx context.Context, id string) (maintenance.Program, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Program{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.programs[id]
	if !ok {
		return maintenance.Program{}, ErrNotFound
	}
	item.Checklist = append([]maintenance.ChecklistItem(nil), item.Checklist...)
	return item, nil
}

func (s *MemoryStore) SaveProgram(ctx context.Context, item maintenance.Program) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item.Checklist = append([]maintenance.ChecklistItem(nil), item.Checklist...)
	s.programs[item.ID] = item
	return nil
}

func (s *MemoryStore) CreatePlan(ctx context.Context, candidate maintenance.Plan) (maintenance.Plan, bool, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.equipment[candidate.EquipmentID]; !ok {
		return maintenance.Plan{}, false, ErrNotFound
	}
	if _, ok := s.programs[candidate.ProgramID]; !ok {
		return maintenance.Plan{}, false, ErrNotFound
	}
	for _, existing := range s.plans {
		if candidate.GenerationKey != "" && existing.GenerationKey == candidate.GenerationKey {
			return existing, false, nil
		}
		if existing.EquipmentID == candidate.EquipmentID && maintenance.BlocksScheduling(existing.Status) && existing.Window.Overlaps(candidate.Window) {
			return maintenance.Plan{}, false, fmt.Errorf("%w: maintenance window overlaps plan %s", ErrConflict, existing.ID)
		}
	}
	s.plans[candidate.ID] = candidate
	return candidate, true, nil
}

func (s *MemoryStore) ListPlans(ctx context.Context) ([]maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]maintenance.Plan, 0, len(s.plans))
	for _, item := range s.plans {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Window.Start.Before(items[j].Window.Start) })
	return items, nil
}

func (s *MemoryStore) GetPlan(ctx context.Context, id string) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.plans[id]
	if !ok {
		return maintenance.Plan{}, ErrNotFound
	}
	return item, nil
}

func (s *MemoryStore) TransitionPlan(ctx context.Context, id string, expectedVersion int64, to maintenance.Status, at time.Time) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.plans[id]
	if !ok {
		return maintenance.Plan{}, ErrNotFound
	}
	if item.Version != expectedVersion {
		return maintenance.Plan{}, ErrVersionConflict
	}
	if err := maintenance.ValidateTransition(item.Status, to); err != nil {
		return maintenance.Plan{}, err
	}
	item.Status = to
	item.Version++
	item.UpdatedAt = at.UTC()
	s.plans[id] = item
	return item, nil
}

func (s *MemoryStore) SaveExecution(ctx context.Context, execution inspection.Execution, findings []inspection.Finding, defects []defect.Defect, expectedVersion int64, at time.Time) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[execution.PlanID]
	if !ok {
		return ErrNotFound
	}
	if plan.Version != expectedVersion {
		return ErrVersionConflict
	}
	if plan.Status != maintenance.StatusInProgress && plan.Status != maintenance.StatusOverdue {
		return fmt.Errorf("%w: plan is not executable", ErrConflict)
	}
	if _, exists := s.executionByPlan[execution.PlanID]; exists {
		return ErrConflict
	}
	execution.Checklist = append([]maintenance.ChecklistItem(nil), execution.Checklist...)
	execution.Measurements = append([]inspection.Measurement(nil), execution.Measurements...)
	s.executions[execution.ID] = execution
	s.executionByPlan[execution.PlanID] = execution.ID
	for _, item := range defects {
		s.defects[item.ID] = item
	}
	plan.Status = maintenance.StatusPendingReview
	plan.Version++
	plan.UpdatedAt = at.UTC()
	s.plans[plan.ID] = plan
	return nil
}

func (s *MemoryStore) GetExecutionByPlan(ctx context.Context, planID string) (inspection.Execution, error) {
	if err := checkContext(ctx); err != nil {
		return inspection.Execution{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.executionByPlan[planID]
	if !ok {
		return inspection.Execution{}, ErrNotFound
	}
	return s.executions[id], nil
}

func (s *MemoryStore) ApplyReview(ctx context.Context, planID, executionID, reviewer, comment string, outcome inspection.ReviewOutcome, expectedVersion int64, restrictionKey string, at time.Time) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[planID]
	if !ok {
		return maintenance.Plan{}, ErrNotFound
	}
	if plan.Version != expectedVersion {
		return maintenance.Plan{}, ErrVersionConflict
	}
	if plan.Status != maintenance.StatusPendingReview {
		return maintenance.Plan{}, ErrConflict
	}
	execution, ok := s.executions[executionID]
	if !ok || execution.PlanID != planID {
		return maintenance.Plan{}, ErrNotFound
	}
	target := maintenance.StatusQualified
	equipmentStatus := equipment.StatusMaintenance
	if outcome == inspection.ReviewFailed {
		target = maintenance.StatusRestricted
		equipmentStatus = equipment.StatusLimited
		plan.EverRestricted = true
		plan.RestrictionKey = restrictionKey
	}
	if err := maintenance.ValidateTransition(plan.Status, target); err != nil {
		return maintenance.Plan{}, err
	}
	execution.Reviewer = reviewer
	execution.ReviewComment = comment
	execution.ReviewOutcome = outcome
	reviewedAt := at.UTC()
	execution.ReviewedAt = &reviewedAt
	s.executions[executionID] = execution
	plan.Status = target
	plan.Version++
	plan.UpdatedAt = reviewedAt
	s.plans[planID] = plan
	eq := s.equipment[plan.EquipmentID]
	eq.Status = equipmentStatus
	eq.Version++
	eq.UpdatedAt = reviewedAt
	s.equipment[eq.ID] = eq
	return plan, nil
}

func (s *MemoryStore) ListDefects(ctx context.Context, equipmentID string) ([]defect.Defect, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]defect.Defect, 0)
	for _, item := range s.defects {
		if equipmentID == "" || item.EquipmentID == equipmentID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DueAt.Before(items[j].DueAt) })
	return items, nil
}

func (s *MemoryStore) SaveRectification(ctx context.Context, item defect.Rectification, expectedVersion int64, at time.Time) (defect.Defect, error) {
	if err := checkContext(ctx); err != nil {
		return defect.Defect{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.defects[item.DefectID]
	if !ok {
		return defect.Defect{}, ErrNotFound
	}
	if current.Version != expectedVersion {
		return defect.Defect{}, ErrVersionConflict
	}
	if current.Status == defect.StatusClosed {
		return defect.Defect{}, ErrConflict
	}
	current.Status = defect.StatusReadyForCheck
	current.Version++
	rectifiedAt := at.UTC()
	current.RectifiedAt = &rectifiedAt
	s.defects[current.ID] = current
	s.rectifications[item.ID] = item
	return current, nil
}

func (s *MemoryStore) SaveReinspection(ctx context.Context, item defect.Reinspection, at time.Time) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[item.PlanID]
	if !ok || plan.EquipmentID != item.EquipmentID {
		return maintenance.Plan{}, ErrNotFound
	}
	if plan.Status != maintenance.StatusRestricted || plan.RestrictionKey != item.RestrictionKey {
		return maintenance.Plan{}, ErrConflict
	}
	for _, id := range item.DefectIDs {
		current, exists := s.defects[id]
		if !exists || current.RestrictionKey != item.RestrictionKey || current.Status != defect.StatusReadyForCheck {
			return maintenance.Plan{}, ErrConflict
		}
	}
	s.reinspections[item.ID] = item
	if item.Passed {
		for _, id := range item.DefectIDs {
			current := s.defects[id]
			current.Status = defect.StatusClosed
			current.Version++
			closedAt := at.UTC()
			current.ClosedAt = &closedAt
			s.defects[id] = current
		}
		plan.Status = maintenance.StatusPendingReview
		plan.Version++
		plan.UpdatedAt = at.UTC()
		s.plans[plan.ID] = plan
	}
	return plan, nil
}

func (s *MemoryStore) MarkReinspectionReviewed(ctx context.Context, reinspectionID string, reviewer string, at time.Time) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.reinspections[reinspectionID]
	if !ok || !item.Passed {
		return maintenance.Plan{}, ErrConflict
	}
	plan := s.plans[item.PlanID]
	if plan.Status != maintenance.StatusPendingReview {
		return maintenance.Plan{}, ErrConflict
	}
	plan.Status = maintenance.StatusQualified
	plan.Version++
	plan.UpdatedAt = at.UTC()
	s.plans[plan.ID] = plan
	reviewedAt := at.UTC()
	item.ReviewedAt = &reviewedAt
	item.Comment = item.Comment + " | reviewer: " + reviewer
	s.reinspections[item.ID] = item
	return plan, nil
}

func (s *MemoryStore) RestoreEquipment(ctx context.Context, planID string, expectedVersion int64, actor, requestID string, at time.Time) (maintenance.Plan, error) {
	if err := checkContext(ctx); err != nil {
		return maintenance.Plan{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[planID]
	if !ok {
		return maintenance.Plan{}, ErrNotFound
	}
	if plan.Version != expectedVersion {
		return maintenance.Plan{}, ErrVersionConflict
	}
	if plan.Status != maintenance.StatusQualified {
		return maintenance.Plan{}, ErrConflict
	}
	if plan.EverRestricted {
		passed := false
		for _, item := range s.reinspections {
			if item.PlanID == planID && item.RestrictionKey == plan.RestrictionKey && item.Passed && item.ReviewedAt != nil {
				passed = true
			}
		}
		if !passed {
			return maintenance.Plan{}, ErrConflict
		}
		for _, item := range s.defects {
			if item.RestrictionKey == plan.RestrictionKey && (item.Level == "critical" || item.Level == "major") && item.Status != defect.StatusClosed {
				return maintenance.Plan{}, ErrConflict
			}
		}
	}
	plan.Status = maintenance.StatusRestored
	plan.Version++
	plan.UpdatedAt = at.UTC()
	s.plans[plan.ID] = plan
	eq := s.equipment[plan.EquipmentID]
	eq.Status = equipment.StatusRunning
	eq.Version++
	eq.UpdatedAt = at.UTC()
	s.equipment[eq.ID] = eq
	s.events = append(s.events, audit.Event{
		ID: fmt.Sprintf("audit-%d", len(s.events)+1), EntityType: "equipment", EntityID: eq.ID,
		Action: "restored", Actor: actor, RequestID: requestID, OccurredAt: at.UTC(),
		Details: map[string]any{"plan_id": plan.ID, "restriction_key": plan.RestrictionKey},
	})
	return plan, nil
}

func (s *MemoryStore) SaveServiceRecord(ctx context.Context, item supplier.ServiceRecord) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.equipment[item.EquipmentID]; !ok {
		return ErrNotFound
	}
	var linkedPlan *maintenance.Plan
	if item.PlanID != "" {
		plan, ok := s.plans[item.PlanID]
		if !ok {
			return ErrNotFound
		}
		linkedPlan = &plan
	}
	if err := supplier.ValidateServiceRecord(item, linkedPlan); err != nil {
		return err
	}
	s.services[item.ID] = item
	return nil
}

func (s *MemoryStore) ListServiceRecords(ctx context.Context, equipmentID string) ([]supplier.ServiceRecord, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]supplier.ServiceRecord, 0)
	for _, item := range s.services {
		if equipmentID == "" || item.EquipmentID == equipmentID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ServicedAt.Before(items[j].ServicedAt) })
	return items, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, event audit.Event) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryStore) ListEvents(ctx context.Context, entityID string) ([]audit.Event, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]audit.Event, 0)
	for _, item := range s.events {
		if entityID == "" || item.EntityID == entityID {
			copyItem := item
			copyItem.Details = make(map[string]any, len(item.Details))
			for key, value := range item.Details {
				copyItem.Details[key] = value
			}
			items = append(items, copyItem)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].OccurredAt.Before(items[j].OccurredAt) })
	return items, nil
}

func (s *MemoryStore) SaveAlert(ctx context.Context, item audit.Alert) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts[item.ID] = item
	return nil
}

func (s *MemoryStore) ListAlerts(ctx context.Context) ([]audit.Alert, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]audit.Alert, 0, len(s.alerts))
	for _, item := range s.alerts {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
