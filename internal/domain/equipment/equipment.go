package equipment

import (
	"errors"
	"strings"
	"time"
)

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

type MaintenanceExposure struct {
	Overdue      bool
	AlertLevel   string
	AlertMessage string
}

var idleOverdueStatuses = map[string]struct {
	level   string
	message string
}{
	"planned": {
		level:   "warning",
		message: "保养计划未在停用窗口内开始",
	},
	"suspended": {
		level:   "warning",
		message: "暂停的保养计划已经超过停用窗口",
	},
}

func AssessMaintenanceExposure(planStatus string, windowEnd, now time.Time) MaintenanceExposure {
	status := normalizeMaintenanceStatus(planStatus)
	if status == "" || !windowHasStrictlyExpired(windowEnd, now) {
		return MaintenanceExposure{}
	}
	rule, eligible := idleOverdueStatuses[status]
	if !eligible {
		return MaintenanceExposure{}
	}
	return idleMaintenanceExposure(rule.level, rule.message)
}

func normalizeMaintenanceStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func windowHasStrictlyExpired(windowEnd, now time.Time) bool {
	if windowEnd.IsZero() || now.IsZero() {
		return false
	}
	return windowEnd.UTC().Before(now.UTC())
}

func idleMaintenanceExposure(level, message string) MaintenanceExposure {
	return MaintenanceExposure{
		Overdue:      true,
		AlertLevel:   level,
		AlertMessage: message,
	}
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
