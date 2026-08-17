package maintenance

import "time"

type ChecklistItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Unit        string   `json:"unit,omitempty"`
	Required    bool     `json:"required"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	EvidenceReq bool     `json:"evidence_required"`
}

type Program struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Version              int64           `json:"version"`
	StatutoryCycleDays   int             `json:"statutory_cycle_days"`
	ApplicableCategories []string        `json:"applicable_categories"`
	Checklist            []ChecklistItem `json:"checklist"`
	Active               bool            `json:"active"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

func (p Program) AppliesTo(category string) bool {
	for _, candidate := range p.ApplicableCategories {
		if candidate == category {
			return true
		}
	}
	return false
}

func (p Program) NextDue(lastQualified time.Time) time.Time {
	return lastQualified.AddDate(0, 0, p.StatutoryCycleDays)
}
