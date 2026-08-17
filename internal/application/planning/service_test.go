package planning

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type sequenceID struct{ value atomic.Int64 }

func (s *sequenceID) New() string { return fmt.Sprintf("id-%d", s.value.Add(1)) }

func planningFixture(t *testing.T) (*Service, *repository.MemoryStore, time.Time) {
	t.Helper()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "b1", Name: "A"}},
		[]equipment.Model{{ID: "m1", Manufacturer: "M", Name: "E", Category: "elevator"}},
		[]equipment.Equipment{{ID: "e1", Code: "E-1", Name: "一号梯", BuildingID: "b1", ModelID: "m1", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now.AddDate(0, 0, -40), UpdatedAt: now}},
		[]maintenance.Program{{ID: "p1", Name: "月检", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator"}, Active: true, UpdatedAt: now}},
	)
	return New(store, &sequenceID{}, clock.Fixed{Time: now}), store, now
}

func TestCreatePlanRejectsConcurrentOverlap(t *testing.T) {
	service, store, now := planningFixture(t)
	const workers = 12
	start := make(chan struct{})
	var wait sync.WaitGroup
	var successes atomic.Int64
	var conflicts atomic.Int64
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, created, err := service.Create(context.Background(), CreatePlan{
				EquipmentID: "e1", ProgramID: "p1", Start: now.Add(time.Hour), End: now.Add(3 * time.Hour),
				Assignee: fmt.Sprintf("worker-%d", index), Actor: "planner",
			})
			if err == nil && created {
				successes.Add(1)
				return
			}
			if errors.Is(err, repository.ErrConflict) {
				conflicts.Add(1)
				return
			}
			t.Errorf("Create() error = %v, created=%v", err, created)
		}(index)
	}
	close(start)
	wait.Wait()
	if successes.Load() != 1 || conflicts.Load() != workers-1 {
		t.Fatalf("successes=%d conflicts=%d", successes.Load(), conflicts.Load())
	}
	plans, err := store.ListPlans(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("stored plans=%d, want 1", len(plans))
	}
	_, created, err := service.Create(context.Background(), CreatePlan{EquipmentID: "e1", ProgramID: "p1", Start: now.Add(3 * time.Hour), End: now.Add(4 * time.Hour), Assignee: "adjacent"})
	if err != nil || !created {
		t.Fatalf("adjacent window should be accepted: created=%v err=%v", created, err)
	}
}

func TestGenerateDuePlansExactlyOnce(t *testing.T) {
	service, store, now := planningFixture(t)
	const workers = 10
	var wait sync.WaitGroup
	start := make(chan struct{})
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			if _, err := service.GenerateDue(context.Background(), now.Add(40*24*time.Hour), "auto"); err != nil {
				t.Errorf("GenerateDue() error = %v", err)
			}
		}()
	}
	close(start)
	wait.Wait()
	plans, err := store.ListPlans(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%d, want exactly one generated cycle", len(plans))
	}
	if plans[0].GenerationKey == "" {
		t.Fatal("generated plan must carry a stable generation key")
	}
}
