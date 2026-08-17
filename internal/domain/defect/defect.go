package defect

import "time"

type Status string

const (
	StatusOpen          Status = "open"
	StatusRectifying    Status = "rectifying"
	StatusReadyForCheck Status = "ready_for_reinspection"
	StatusClosed        Status = "closed"
)

type Defect struct {
	ID             string     `json:"id"`
	EquipmentID    string     `json:"equipment_id"`
	PlanID         string     `json:"plan_id"`
	ExecutionID    string     `json:"execution_id"`
	ChecklistItem  string     `json:"checklist_item_id"`
	Level          string     `json:"level"`
	Description    string     `json:"description"`
	Status         Status     `json:"status"`
	DueAt          time.Time  `json:"due_at"`
	RectifiedAt    *time.Time `json:"rectified_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	Version        int64      `json:"version"`
	RestrictionKey string     `json:"restriction_key"`
}

type Rectification struct {
	ID         string    `json:"id"`
	DefectID   string    `json:"defect_id"`
	Action     string    `json:"action"`
	Operator   string    `json:"operator"`
	EvidenceID string    `json:"evidence_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Reinspection struct {
	ID             string     `json:"id"`
	DefectIDs      []string   `json:"defect_ids"`
	PlanID         string     `json:"plan_id"`
	EquipmentID    string     `json:"equipment_id"`
	RestrictionKey string     `json:"restriction_key"`
	Inspector      string     `json:"inspector"`
	Passed         bool       `json:"passed"`
	Comment        string     `json:"comment"`
	InspectedAt    time.Time  `json:"inspected_at"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
}

func BlocksRestoration(defects []Defect) bool {
	for _, item := range defects {
		if (item.Level == "critical" || item.Level == "major") && item.Status != StatusClosed {
			return true
		}
	}
	return false
}
