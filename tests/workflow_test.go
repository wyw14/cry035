package tests

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/application/review"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type ids struct{ value atomic.Int64 }

func (i *ids) New() string { return fmt.Sprintf("workflow-%d", i.value.Add(1)) }

func TestRestrictedEquipmentRequiresReviewedPassingReinspection(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "b1", Name: "安全中心"}},
		[]equipment.Model{{ID: "m1", Manufacturer: "厂商", Name: "型号", Category: "elevator"}},
		[]equipment.Equipment{{ID: "e1", Code: "E1", Name: "客梯", BuildingID: "b1", ModelID: "m1", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{ID: "p1", Name: "制动月检", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator"}, Checklist: []maintenance.ChecklistItem{{ID: "brake", Name: "制动器", Required: true}}, Active: true, UpdatedAt: now}},
	)
	idSource := &ids{}
	fixedClock := clock.Fixed{Time: now}
	planningService := planning.New(store, idSource, fixedClock)
	executionService := execution.New(store, idSource, fixedClock)
	reviewService := review.New(store, idSource, fixedClock)
	remediationService := remediation.New(store, idSource, fixedClock)

	plan, _, err := planningService.Create(ctx, planning.CreatePlan{EquipmentID: "e1", ProgramID: "p1", Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Assignee: "tech", Actor: "planner"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = planningService.Transition(ctx, plan.ID, plan.Version, maintenance.StatusInProgress, "tech", "req-start")
	if err != nil {
		t.Fatal(err)
	}
	_, defects, err := executionService.Submit(ctx, execution.Submit{PlanID: plan.ID, ExpectedVersion: plan.Version, Technician: "tech", Measurements: []inspection.Measurement{{ItemID: "brake", Passed: false, Notes: "制动力不足"}}, RequestID: "req-execute"})
	if err != nil {
		t.Fatal(err)
	}
	if len(defects) != 1 {
		t.Fatalf("defects=%d, want 1", len(defects))
	}
	plan, err = store.GetPlan(ctx, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	plan, err = reviewService.Decide(ctx, review.Decide{PlanID: plan.ID, ExpectedVersion: plan.Version, Reviewer: "reviewer", Outcome: inspection.ReviewFailed, Comment: "需整改", RequestID: "req-review"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != maintenance.StatusRestricted {
		t.Fatalf("status=%s", plan.Status)
	}
	if _, err := remediationService.Restore(ctx, plan.ID, plan.Version, "manager", "req-direct-restore"); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("direct restore error=%v, want conflict", err)
	}
	updatedDefect, err := remediationService.Rectify(ctx, remediation.Rectify{DefectID: defects[0].ID, ExpectedVersion: defects[0].Version, Action: "更换制动衬片", Operator: "tech", RequestID: "req-rectify"})
	if err != nil {
		t.Fatal(err)
	}
	reinspection, plan, err := remediationService.Reinspect(ctx, remediation.Reinspect{PlanID: plan.ID, EquipmentID: "e1", DefectIDs: []string{updatedDefect.ID}, RestrictionKey: plan.RestrictionKey, Inspector: "reviewer", Passed: true, Comment: "复检合格", RequestID: "req-reinspect"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := remediationService.Restore(ctx, plan.ID, plan.Version, "manager", "req-unreviewed"); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("unreviewed reinspection restore error=%v, want conflict", err)
	}
	plan, err = remediationService.ReviewReinspection(ctx, reinspection.ID, "safety-manager", "req-reinspect-review")
	if err != nil {
		t.Fatal(err)
	}
	plan, err = remediationService.Restore(ctx, plan.ID, plan.Version, "safety-manager", "req-restore")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != maintenance.StatusRestored {
		t.Fatalf("status=%s, want restored", plan.Status)
	}
	equipmentItem, err := store.GetEquipment(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if equipmentItem.Status != equipment.StatusRunning {
		t.Fatalf("equipment status=%s", equipmentItem.Status)
	}
}
