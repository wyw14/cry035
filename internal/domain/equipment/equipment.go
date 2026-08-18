package equipment

import (
	"errors"
	"strings"
	"time"
)

func AlertDeliveryKey(alertID string) string { return "equipment-alert:" + alertID }

type OperatingStatus string

const (
	StatusRunning     OperatingStatus = "running"
	StatusMaintenance OperatingStatus = "maintenance"
	StatusLimited     OperatingStatus = "limited"
	StatusStopped     OperatingStatus = "stopped"
)

var ErrInvalidEquipment = errors.New("invalid equipment")

type Building struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Model struct {
	ID           string `json:"id"`
	Manufacturer string `json:"manufacturer"`
	Name         string `json:"name"`
	Category     string `json:"category"`
}

type Equipment struct {
	ID              string          `json:"id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	BuildingID      string          `json:"building_id"`
	ModelID         string          `json:"model_id"`
	Location        string          `json:"location"`
	ResponsibleUnit string          `json:"responsible_unit"`
	Status          OperatingStatus `json:"status"`
	Version         int64           `json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func (e Equipment) Validate() error {
	if strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Code) == "" || strings.TrimSpace(e.Name) == "" {
		return ErrInvalidEquipment
	}
	if e.BuildingID == "" || e.ModelID == "" || e.ResponsibleUnit == "" {
		return ErrInvalidEquipment
	}
	switch e.Status {
	case StatusRunning, StatusMaintenance, StatusLimited, StatusStopped:
		return nil
	default:
		return ErrInvalidEquipment
	}
}
