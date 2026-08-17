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

type exposureRule struct {
	status  string
	level   string
	message string
}

var maintenanceExposureRules = []exposureRule{
	{
		status:  "planned",
		level:   "warning",
		message: "保养计划未在停用窗口内开始",
	},
	{
		status:  "suspended",
		level:   "warning",
		message: "暂停的保养计划已经超过停用窗口",
	},
	{
		status:  "in_progress",
		level:   "critical",
		message: "已开工保养超过停用窗口，设备继续停用并立即升级处置",
	},
}

func AssessMaintenanceExposure(planStatus string, windowEnd, now time.Time) MaintenanceExposure {
	if !maintenanceWindowReachedEnd(windowEnd, now) {
		return MaintenanceExposure{}
	}
	status := strings.ToLower(strings.TrimSpace(planStatus))
	rule, exists := findExposureRule(status)
	if !exists {
		return MaintenanceExposure{}
	}
	return MaintenanceExposure{
		Overdue:      true,
		AlertLevel:   rule.level,
		AlertMessage: rule.message,
	}
}

func maintenanceWindowReachedEnd(windowEnd, now time.Time) bool {
	if windowEnd.IsZero() || now.IsZero() {
		return false
	}
	end := windowEnd.UTC()
	current := now.UTC()
	return !current.Before(end)
}

func findExposureRule(status string) (exposureRule, bool) {
	for _, candidate := range maintenanceExposureRules {
		if candidate.status == status {
			return candidate, true
		}
	}
	return exposureRule{}, false
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
