package inspection

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type Evidence struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	StoredPath  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

type Measurement struct {
	ItemID     string   `json:"item_id"`
	Value      *float64 `json:"value,omitempty"`
	Passed     bool     `json:"passed"`
	Notes      string   `json:"notes,omitempty"`
	EvidenceID string   `json:"evidence_id,omitempty"`
}

type Execution struct {
	ID            string                      `json:"id"`
	PlanID        string                      `json:"plan_id"`
	Technician    string                      `json:"technician"`
	Checklist     []maintenance.ChecklistItem `json:"checklist_snapshot"`
	Measurements  []Measurement               `json:"measurements"`
	Evidence      []Evidence                  `json:"evidence"`
	SubmittedAt   time.Time                   `json:"submitted_at"`
	ReviewComment string                      `json:"review_comment,omitempty"`
	ReviewOutcome ReviewOutcome               `json:"review_outcome,omitempty"`
	Reviewer      string                      `json:"reviewer,omitempty"`
	ReviewedAt    *time.Time                  `json:"reviewed_at,omitempty"`
}

type ReviewOutcome string

const (
	ReviewPassed ReviewOutcome = "passed"
	ReviewFailed ReviewOutcome = "failed"
)

var (
	ErrIncompleteChecklist  = errors.New("required checklist item is missing")
	ErrEvidenceRequired     = errors.New("required photo evidence is missing")
	ErrDuplicateEvidence    = errors.New("duplicate execution evidence id")
	ErrDuplicateMeasurement = errors.New("duplicate checklist measurement")
	ErrEvidenceReused       = errors.New("photo evidence is assigned to multiple checklist items")
	ErrInvalidEvidence      = errors.New("invalid execution evidence")
)

type submissionIndex struct {
	evidenceByID map[string]Evidence
	measured     map[string]Measurement
}

func newSubmissionIndex(measurements []Measurement, evidence []Evidence) submissionIndex {
	index := submissionIndex{
		evidenceByID: make(map[string]Evidence, len(evidence)),
		measured:     make(map[string]Measurement, len(measurements)),
	}
	for _, item := range evidence {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		item.ID = id
		index.evidenceByID[id] = item
	}
	for _, measurement := range measurements {
		itemID := strings.TrimSpace(measurement.ItemID)
		if itemID == "" {
			continue
		}
		measurement.ItemID = itemID
		index.measured[itemID] = measurement
	}
	return index
}

func ValidateSubmission(checklist []maintenance.ChecklistItem, measurements []Measurement, evidence []Evidence) error {
	index := newSubmissionIndex(measurements, evidence)
	for _, measurement := range measurements {
		if measurement.EvidenceID == "" {
			continue
		}
		if _, ok := index.evidenceByID[measurement.EvidenceID]; !ok {
			return fmt.Errorf("%w: %s", ErrInvalidEvidence, measurement.EvidenceID)
		}
	}
	for _, item := range checklist {
		measurement, exists := index.measured[item.ID]
		if !exists || !item.EvidenceReq {
			continue
		}
		if measurement.EvidenceID == "" {
			return fmt.Errorf("%w: %s", ErrEvidenceRequired, item.ID)
		}
	}
	return nil
}

type Finding struct {
	ItemID string `json:"item_id"`
	Level  string `json:"level"`
	Reason string `json:"reason"`
}

func Evaluate(checklist []maintenance.ChecklistItem, measurements []Measurement) ([]Finding, error) {
	byItem := make(map[string]Measurement, len(measurements))
	for _, measurement := range measurements {
		byItem[measurement.ItemID] = measurement
	}
	findings := make([]Finding, 0)
	for _, item := range checklist {
		measurement, exists := byItem[item.ID]
		if item.Required && !exists {
			return nil, fmt.Errorf("%w: %s", ErrIncompleteChecklist, item.ID)
		}
		if !exists {
			continue
		}
		if item.EvidenceReq && measurement.EvidenceID == "" {
			return nil, fmt.Errorf("%w: %s", ErrEvidenceRequired, item.ID)
		}
		passed := measurement.Passed
		if measurement.Value != nil {
			if item.Minimum != nil && *measurement.Value < *item.Minimum {
				passed = false
			}
			if item.Maximum != nil && *measurement.Value > *item.Maximum {
				passed = false
			}
		}
		if !passed {
			level := "major"
			if item.Required {
				level = "critical"
			}
			findings = append(findings, Finding{ItemID: item.ID, Level: level, Reason: measurement.Notes})
		}
	}
	return findings, nil
}
