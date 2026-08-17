package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type evidenceIDs002 struct{ next atomic.Int64 }

func (i *evidenceIDs002) New() string { return fmt.Sprintf("evidence-002-%d", i.next.Add(1)) }

func TestExecutionRejectsReusedChecklistEvidence002(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "building-002", Name: "乙楼"}},
		[]equipment.Model{{ID: "model-002", Name: "货梯", Category: "freight"}},
		[]equipment.Equipment{{ID: "equipment-002", Code: "FT-002", Name: "二号货梯", BuildingID: "building-002", ModelID: "model-002", ResponsibleUnit: "设备部", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{
			ID: "program-002", Name: "门锁与制动检查", Version: 1, StatutoryCycleDays: 30,
			ApplicableCategories: []string{"freight"}, Active: true,
			Checklist: []maintenance.ChecklistItem{
				{ID: "door-lock-002", Name: "门锁", Required: true, EvidenceReq: true},
				{ID: "brake-002", Name: "制动器", Required: true, EvidenceReq: true},
			},
		}},
	)
	idSource := &evidenceIDs002{}
	fixedClock := clock.Fixed{Time: now}
	planningService := planning.New(store, idSource, fixedClock)
	plan, _, err := planningService.Create(ctx, planning.CreatePlan{EquipmentID: "equipment-002", ProgramID: "program-002", Start: now.Add(time.Hour), End: now.Add(3 * time.Hour), Assignee: "tech-002"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = planningService.Transition(ctx, plan.ID, plan.Version, maintenance.StatusInProgress, "tech-002", "start-002")
	if err != nil {
		t.Fatal(err)
	}
	service := execution.New(store, idSource, fixedClock)
	_, _, err = service.Submit(ctx, execution.Submit{
		PlanID: plan.ID, ExpectedVersion: plan.Version, Technician: "tech-002",
		Evidence: []inspection.Evidence{{ID: "photo-002", FileName: "inspection.jpg", ContentType: "image/jpeg", Size: 2048, SHA256: strings.Repeat("a", 64)}},
		Measurements: []inspection.Measurement{
			{ItemID: "door-lock-002", Passed: true, EvidenceID: "photo-002"},
			{ItemID: "brake-002", Passed: true, EvidenceID: "photo-002"},
		},
	})
	if !errors.Is(err, inspection.ErrEvidenceReused) {
		t.Fatalf("Submit() error = %v, want evidence reuse rejection", err)
	}
	if _, lookupErr := store.GetExecutionByPlan(ctx, plan.ID); !errors.Is(lookupErr, repository.ErrNotFound) {
		t.Fatalf("execution persisted after rejected evidence: %v", lookupErr)
	}
}
