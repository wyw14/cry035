package tests

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/domain/supplier"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type ledgerIDs005 struct{ next atomic.Int64 }

func (i *ledgerIDs005) New() string { return fmt.Sprintf("ledger-005-%d", i.next.Add(1)) }

func TestSupplierLedgerRejectsCrossEquipmentPlan005(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "building-005", Name: "戊楼"}},
		[]equipment.Model{{ID: "model-005", Name: "客梯", Category: "elevator"}},
		[]equipment.Equipment{
			{ID: "equipment-005-a", Code: "EL-005-A", Name: "A梯", BuildingID: "building-005", ModelID: "model-005", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now},
			{ID: "equipment-005-b", Code: "EL-005-B", Name: "B梯", BuildingID: "building-005", ModelID: "model-005", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now},
		},
		[]maintenance.Program{{ID: "program-005", Name: "季度维保", Version: 1, StatutoryCycleDays: 90, ApplicableCategories: []string{"elevator"}, Active: true}},
	)
	plan, created, err := store.CreatePlan(ctx, maintenance.Plan{ID: "plan-005-b", EquipmentID: "equipment-005-b", ProgramID: "program-005", ProgramVersion: 1, Window: maintenance.Window{Start: now, End: now.Add(2 * time.Hour)}, Assignee: "vendor-tech", Status: maintenance.StatusQualified, Version: 1, CreatedAt: now, UpdatedAt: now})
	if err != nil || !created {
		t.Fatalf("CreatePlan() created=%v err=%v", created, err)
	}
	service := reporting.New(store, &ledgerIDs005{}, clock.Fixed{Time: now})
	_, err = service.RecordService(ctx, reporting.RecordService{
		VendorName: "本地维保公司", EquipmentID: "equipment-005-a", PlanID: plan.ID,
		Description: "季度保养", AmountCents: 36000, Currency: "CNY", ServicedAt: now,
		Actor: "finance-005", RequestID: "ledger-request-005",
	})
	if !errors.Is(err, supplier.ErrPlanEquipmentMismatch) {
		t.Fatalf("RecordService() error = %v, want cross-equipment plan rejection", err)
	}
	history, historyErr := service.History(ctx, "equipment-005-a")
	if historyErr != nil {
		t.Fatal(historyErr)
	}
	if len(history.Services) != 0 || history.CostTotal != 0 {
		t.Fatalf("rejected service leaked into ledger: services=%d total=%d", len(history.Services), history.CostTotal)
	}
}
