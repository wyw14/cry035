package audit

import "time"

type Event struct {
	ID         string         `json:"id"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Action     string         `json:"action"`
	Actor      string         `json:"actor"`
	RequestID  string         `json:"request_id"`
	Details    map[string]any `json:"details"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type Alert struct {
	ID          string    `json:"id"`
	EquipmentID string    `json:"equipment_id"`
	PlanID      string    `json:"plan_id"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Read        bool      `json:"read"`
	CreatedAt   time.Time `json:"created_at"`
}
