package repository

import (
	"context"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/domain/supplier"
)

// Store is the composition root contract. Application packages depend on smaller local interfaces.
type Store interface {
	ListBuildings(context.Context) ([]equipment.Building, error)
	ListEquipment(context.Context) ([]equipment.Equipment, error)
	GetEquipment(context.Context, string) (equipment.Equipment, error)
	GetModel(context.Context, string) (equipment.Model, error)
	SaveEquipment(context.Context, equipment.Equipment) error
	ListPrograms(context.Context) ([]maintenance.Program, error)
	GetProgram(context.Context, string) (maintenance.Program, error)
	SaveProgram(context.Context, maintenance.Program) error
	CreatePlan(context.Context, maintenance.Plan) (maintenance.Plan, bool, error)
	ListPlans(context.Context) ([]maintenance.Plan, error)
	GetPlan(context.Context, string) (maintenance.Plan, error)
	TransitionPlan(context.Context, string, int64, maintenance.Status, time.Time) (maintenance.Plan, error)
	SaveExecution(context.Context, inspection.Execution, []inspection.Finding, []defect.Defect, int64, time.Time) error
	GetExecutionByPlan(context.Context, string) (inspection.Execution, error)
	ApplyReview(context.Context, string, string, string, string, inspection.ReviewOutcome, int64, string, time.Time) (maintenance.Plan, error)
	ListDefects(context.Context, string) ([]defect.Defect, error)
	SaveRectification(context.Context, defect.Rectification, int64, time.Time) (defect.Defect, error)
	SaveReinspection(context.Context, defect.Reinspection, time.Time) (maintenance.Plan, error)
	MarkReinspectionReviewed(context.Context, string, string, time.Time) (maintenance.Plan, error)
	RestoreEquipment(context.Context, string, int64, string, string, time.Time) (maintenance.Plan, error)
	SaveServiceRecord(context.Context, supplier.ServiceRecord) error
	ListServiceRecords(context.Context, string) ([]supplier.ServiceRecord, error)
	AppendEvent(context.Context, audit.Event) error
	ListEvents(context.Context, string) ([]audit.Event, error)
	SaveAlert(context.Context, audit.Alert) error
	ListAlerts(context.Context) ([]audit.Alert, error)
}
