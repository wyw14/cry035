package tests

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type generationIDs001 struct{ next atomic.Int64 }

func (i *generationIDs001) New() string { return fmt.Sprintf("generation-001-%d", i.next.Add(1)) }

func TestAutomaticGenerationAllocatesDistinctShutdownWindows001(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "building-001", Name: "甲楼"}},
		[]equipment.Model{{ID: "model-001", Name: "客梯", Category: "elevator"}},
		[]equipment.Equipment{{
			ID: "equipment-001", Code: "EL-001", Name: "一号客梯", BuildingID: "building-001",
			ModelID: "model-001", ResponsibleUnit: "物业工程部", Status: equipment.StatusRunning,
			Version: 1, CreatedAt: now.AddDate(0, 0, -40), UpdatedAt: now,
		}},
		[]maintenance.Program{
			{ID: "program-brake-001", Name: "制动检查", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator"}, Active: true},
			{ID: "program-door-001", Name: "层门检查", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator"}, Active: true},
		},
	)
	service := planning.New(store, &generationIDs001{}, clock.Fixed{Time: now})
	created, err := service.GenerateDue(ctx, now.Add(25*24*time.Hour), "自动排班")
	if err != nil {
		t.Fatalf("GenerateDue() error = %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created plans = %d, want 2", len(created))
	}
	if created[0].EquipmentID != created[1].EquipmentID {
		t.Fatalf("equipment ids = %q and %q", created[0].EquipmentID, created[1].EquipmentID)
	}
	if created[0].Window.Overlaps(created[1].Window) {
		t.Fatalf("generated shutdown windows overlap: %#v and %#v", created[0].Window, created[1].Window)
	}
	if !created[0].Window.End.Equal(created[1].Window.Start) {
		t.Fatalf("windows should be packed adjacently: %#v and %#v", created[0].Window, created[1].Window)
	}
}
