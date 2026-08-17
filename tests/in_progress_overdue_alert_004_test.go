package tests

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
	"github.com/wyw14/cry035/internal/service/scheduler"
)

type overdueIDs004 struct{ next atomic.Int64 }

func (i *overdueIDs004) New() string { return fmt.Sprintf("overdue-004-%d", i.next.Add(1)) }

func TestInProgressPlanBecomesOverdueAndAlerts004(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 16, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "building-004", Name: "丁楼"}},
		[]equipment.Model{{ID: "model-004", Name: "升降机", Category: "lift"}},
		[]equipment.Equipment{{ID: "equipment-004", Code: "LF-004", Name: "四号升降机", BuildingID: "building-004", ModelID: "model-004", ResponsibleUnit: "工程部", Status: equipment.StatusMaintenance, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{ID: "program-004", Name: "年度检查", Version: 1, StatutoryCycleDays: 365, ApplicableCategories: []string{"lift"}, Active: true}},
	)
	ids := &overdueIDs004{}
	fixedClock := clock.Fixed{Time: now}
	planner := planning.New(store, ids, fixedClock)
	plan, _, err := planner.Create(ctx, planning.CreatePlan{EquipmentID: "equipment-004", ProgramID: "program-004", Start: now.Add(-3 * time.Hour), End: now.Add(-time.Hour), Assignee: "tech-004"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = planner.Transition(ctx, plan.ID, plan.Version, maintenance.StatusInProgress, "tech-004", "start-004")
	if err != nil {
		t.Fatal(err)
	}
	worker := scheduler.New(planner, store, ids, fixedClock)
	if _, err := worker.RunOnce(ctx, now); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	updated, err := store.GetPlan(ctx, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != maintenance.StatusOverdue {
		t.Fatalf("plan status = %s, want overdue", updated.Status)
	}
	alerts, err := store.ListAlerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Level != "critical" || !strings.Contains(alerts[0].Message, "已开工") {
		t.Fatalf("alerts = %#v, want one critical in-progress overrun alert", alerts)
	}
}
