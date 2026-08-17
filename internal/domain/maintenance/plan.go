package maintenance

import "time"

type SpareRequirement struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type Plan struct {
	ID             string             `json:"id"`
	EquipmentID    string             `json:"equipment_id"`
	ProgramID      string             `json:"program_id"`
	ProgramVersion int64              `json:"program_version"`
	GenerationKey  string             `json:"generation_key"`
	Window         Window             `json:"window"`
	Assignee       string             `json:"assignee"`
	Spares         []SpareRequirement `json:"spares"`
	Status         Status             `json:"status"`
	EverRestricted bool               `json:"ever_restricted"`
	RestrictionKey string             `json:"restriction_key,omitempty"`
	Version        int64              `json:"version"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}
