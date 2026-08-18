package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type reviewRepo014 struct {
	reviewedAt time.Time
	event      audit.Event
}

func (r *reviewRepo014) GetPlan(context.Context, string) (maintenance.Plan, error) {
	return maintenance.Plan{}, nil
}
func (r *reviewRepo014) ListDefects(context.Context, string) ([]defect.Defect, error) {
	return nil, nil
}
func (r *reviewRepo014) SaveRectification(context.Context, defect.Rectification, int64, time.Time) (defect.Defect, error) {
	return defect.Defect{}, nil
}
func (r *reviewRepo014) SaveReinspection(context.Context, defect.Reinspection, time.Time) (maintenance.Plan, error) {
	return maintenance.Plan{}, nil
}
func (r *reviewRepo014) MarkReinspectionReviewed(_ context.Context, _ string, _ string, at time.Time) (maintenance.Plan, error) {
	r.reviewedAt = at
	return maintenance.Plan{ID: "plan-014", EquipmentID: "equipment-014"}, nil
}
func (r *reviewRepo014) RestoreEquipment(context.Context, string, int64, string, string, time.Time) (maintenance.Plan, error) {
	return maintenance.Plan{}, nil
}
func (r *reviewRepo014) AppendEvent(_ context.Context, event audit.Event) error {
	r.event = event
	return nil
}

type advancingClock014 struct{ calls int }

func (c *advancingClock014) Now() time.Time {
	c.calls++
	return time.Date(2026, 8, 18, 10, 0, c.calls*5, 0, time.UTC)
}

type fixedID014 struct{}

func (fixedID014) New() string { return "audit-014" }

func TestReviewReinspectionUsesSingleAuditTimestamp014(t *testing.T) {
	repo := &reviewRepo014{}
	service := New(repo, fixedID014{}, &advancingClock014{})
	if _, err := service.ReviewReinspection(context.Background(), "reinspection-014", "reviewer-014", "request-014"); err != nil {
		t.Fatal(err)
	}
	if !repo.reviewedAt.Equal(repo.event.OccurredAt) {
		t.Fatalf("reviewed_at=%s event_at=%s", repo.reviewedAt, repo.event.OccurredAt)
	}
}
