package inspection

import (
	"errors"
	"testing"

	"github.com/wyw14/cry035/internal/domain/maintenance"
)

func TestEvaluateRequiresCompleteChecklist(t *testing.T) {
	items := []maintenance.ChecklistItem{
		{ID: "door-lock", Name: "层门门锁", Required: true},
		{ID: "lamp", Name: "轿厢照明"},
	}
	_, err := Evaluate(items, []Measurement{{ItemID: "lamp", Passed: true}})
	if !errors.Is(err, ErrIncompleteChecklist) {
		t.Fatalf("Evaluate() error = %v, want incomplete checklist", err)
	}
}

func TestEvaluateProducesCriticalFinding(t *testing.T) {
	items := []maintenance.ChecklistItem{{ID: "brake", Name: "制动器", Required: true}}
	findings, err := Evaluate(items, []Measurement{{ItemID: "brake", Passed: false, Notes: "制动力不足"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Level != "critical" {
		t.Fatalf("findings = %#v", findings)
	}
}
