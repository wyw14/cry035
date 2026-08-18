package tests

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/repository"
)

func TestPlanSpareRequirementsAreSnapshotIsolated007(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(nil,
		[]equipment.Model{{ID: "model-007", Category: "elevator"}},
		[]equipment.Equipment{{ID: "equipment-007", Code: "E-007", ModelID: "model-007", Status: equipment.StatusRunning, Version: 1}},
		[]maintenance.Program{{ID: "program-007", Version: 3, Active: true}},
	)
	window, err := maintenance.NewWindow(now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	spares := []maintenance.SpareRequirement{{Name: "brake-pad", Quantity: 2}}
	created, ok, err := store.CreatePlan(ctx, maintenance.Plan{
		ID: "plan-007", EquipmentID: "equipment-007", ProgramID: "program-007", ProgramVersion: 3,
		Window: window, Spares: spares, Status: maintenance.StatusPlanned, Version: 1,
	})
	if err != nil || !ok {
		t.Fatalf("create plan: created=%v err=%v", ok, err)
	}
	spares[0].Quantity = 99
	created.Spares[0].Name = "external-change"

	first, err := store.GetPlan(ctx, "plan-007")
	if err != nil {
		t.Fatal(err)
	}
	if first.Spares[0].Quantity != 2 || first.Spares[0].Name != "brake-pad" {
		t.Fatalf("stored spares changed through create aliases: %+v", first.Spares)
	}
	first.Spares[0].Quantity = 77
	second, err := store.GetPlan(ctx, "plan-007")
	if err != nil {
		t.Fatal(err)
	}
	if second.Spares[0].Quantity != 2 {
		t.Fatalf("stored spares changed through read alias: %+v", second.Spares)
	}
}
