package tests

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

func TestAuditHistorySnapshotIsolation006(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()
	store.Seed(nil, nil, []equipment.Equipment{{ID: "e-006", Code: "E-006", Status: equipment.StatusRunning, Version: 1}}, nil)
	err := store.AppendEvent(ctx, audit.Event{
		ID: "event-006", EntityType: "equipment", EntityID: "e-006", Action: "inspection_recorded",
		OccurredAt: time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC),
		Details: map[string]any{
			"evidence": []string{"before.jpg", "after.jpg"},
			"review":   map[string]string{"result": "passed", "actor": "reviewer-a"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	service := reporting.New(store, &ids{}, clock.Fixed{Time: time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)})
	firstHistory, err := service.History(ctx, "e-006")
	if err != nil || len(firstHistory.Events) != 1 {
		t.Fatalf("first read: len=%d err=%v", len(firstHistory.Events), err)
	}
	firstHistory.Events[0].Details["evidence"].([]string)[0] = "replaced.jpg"
	firstHistory.Events[0].Details["review"].(map[string]string)["actor"] = "outsider"

	secondHistory, err := service.History(ctx, "e-006")
	if err != nil || len(secondHistory.Events) != 1 {
		t.Fatalf("second read: len=%d err=%v", len(secondHistory.Events), err)
	}
	if got := secondHistory.Events[0].Details["evidence"].([]string)[0]; got != "before.jpg" {
		t.Fatalf("stored evidence changed through read result: %q", got)
	}
	if got := secondHistory.Events[0].Details["review"].(map[string]string)["actor"]; got != "reviewer-a" {
		t.Fatalf("stored review changed through read result: %q", got)
	}
}
