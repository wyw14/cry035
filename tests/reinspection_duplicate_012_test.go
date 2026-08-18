package tests

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/clock"
)

type reinspectionRepo012 struct{}

func (reinspectionRepo012) GetPlan(context.Context, string) (maintenance.Plan, error) {
	return maintenance.Plan{ID: "plan-012", EquipmentID: "equipment-012"}, nil
}
func (reinspectionRepo012) ListDefects(context.Context, string) ([]defect.Defect, error) {
	return nil, nil
}
func (reinspectionRepo012) SaveRectification(context.Context, defect.Rectification, int64, time.Time) (defect.Defect, error) {
	return defect.Defect{}, nil
}
func (reinspectionRepo012) SaveReinspection(_ context.Context, item defect.Reinspection, _ time.Time) (maintenance.Plan, error) {
	return maintenance.Plan{ID: item.PlanID, EquipmentID: item.EquipmentID}, nil
}
func (reinspectionRepo012) MarkReinspectionReviewed(context.Context, string, string, time.Time) (maintenance.Plan, error) {
	return maintenance.Plan{}, nil
}
func (reinspectionRepo012) RestoreEquipment(context.Context, string, int64, string, string, time.Time) (maintenance.Plan, error) {
	return maintenance.Plan{}, nil
}
func (reinspectionRepo012) AppendEvent(context.Context, audit.Event) error { return nil }

func TestReinspectionRejectsDuplicateDefectIDs012(t *testing.T) {
	service := remediation.New(reinspectionRepo012{}, &ids{}, clock.Fixed{Time: time.Date(2026, 8, 18, 15, 0, 0, 0, time.UTC)})
	_, _, err := service.Reinspect(context.Background(), remediation.Reinspect{PlanID: "plan-012", EquipmentID: "equipment-012", DefectIDs: []string{"defect-012", "defect-012"}, RestrictionKey: "restriction-012", Inspector: "safety-012", Passed: true})
	if err == nil {
		t.Fatalf("duplicate defect IDs error = %v, want rejection", err)
	}
}
