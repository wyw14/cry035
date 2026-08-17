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
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type restorationIDs003 struct{ next atomic.Int64 }

func (i *restorationIDs003) New() string { return fmt.Sprintf("restore-003-%d", i.next.Add(1)) }

func TestRestoreRequiresEveryBlockingDefectClosed003(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "building-003", Name: "丙楼"}},
		[]equipment.Model{{ID: "model-003", Name: "客梯", Category: "elevator"}},
		[]equipment.Equipment{{ID: "equipment-003", Code: "EL-003", Name: "三号客梯", BuildingID: "building-003", ModelID: "model-003", ResponsibleUnit: "安全部", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{
			ID: "program-003", Name: "双项安全检查", Version: 1, StatutoryCycleDays: 30,
			ApplicableCategories: []string{"elevator"}, Active: true,
			Checklist: []maintenance.ChecklistItem{
				{ID: "brake-003", Name: "制动器", Required: true},
				{ID: "door-003", Name: "层门门锁", Required: true},
			},
		}},
	)
	ids := &restorationIDs003{}
	fixedClock := clock.Fixed{Time: now}
	planner := planning.New(store, ids, fixedClock)
	executor := execution.New(store, ids, fixedClock)
	reviewer := review.New(store, ids, fixedClock)
	remediator := remediation.New(store, ids, fixedClock)

	plan, _, err := planner.Create(ctx, planning.CreatePlan{EquipmentID: "equipment-003", ProgramID: "program-003", Start: now.Add(time.Hour), End: now.Add(3 * time.Hour), Assignee: "tech-003"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = planner.Transition(ctx, plan.ID, plan.Version, maintenance.StatusInProgress, "tech-003", "start-003")
	if err != nil {
		t.Fatal(err)
	}
	_, defects, err := executor.Submit(ctx, execution.Submit{
		PlanID: plan.ID, ExpectedVersion: plan.Version, Technician: "tech-003",
		Measurements: []inspection.Measurement{
			{ItemID: "brake-003", Passed: false, Notes: "制动力不足"},
			{ItemID: "door-003", Passed: false, Notes: "门锁触点失效"},
		},
	})
	if err != nil || len(defects) != 2 {
		t.Fatalf("Submit() defects=%d err=%v", len(defects), err)
	}
	plan, _ = store.GetPlan(ctx, plan.ID)
	plan, err = reviewer.Decide(ctx, review.Decide{PlanID: plan.ID, ExpectedVersion: plan.Version, Reviewer: "reviewer-003", Outcome: inspection.ReviewFailed, Comment: "两项均需整改"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := remediator.Rectify(ctx, remediation.Rectify{DefectID: defects[0].ID, ExpectedVersion: defects[0].Version, Action: "更换部件", Operator: "tech-003"})
	if err != nil {
		t.Fatal(err)
	}
	reinspection, plan, err := remediator.Reinspect(ctx, remediation.Reinspect{PlanID: plan.ID, EquipmentID: plan.EquipmentID, DefectIDs: []string{first.ID}, RestrictionKey: plan.RestrictionKey, Inspector: "inspector-003", Passed: true, Comment: "单项复检合格"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = remediator.ReviewReinspection(ctx, reinspection.ID, "safety-003", "review-003")
	if err != nil {
		t.Fatal(err)
	}
	_, err = remediator.Restore(ctx, plan.ID, plan.Version, "safety-003", "restore-003")
	if !errors.Is(err, defect.ErrRestorationBlocked) {
		t.Fatalf("Restore() error = %v, want remaining blocking defect", err)
	}
	equipmentItem, lookupErr := store.GetEquipment(ctx, "equipment-003")
	if lookupErr != nil {
		t.Fatal(lookupErr)
	}
	if equipmentItem.Status != equipment.StatusLimited {
		t.Fatalf("equipment status = %s, want limited", equipmentItem.Status)
	}
}
