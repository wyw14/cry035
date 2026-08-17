package supplier

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wyw14/cry035/internal/domain/maintenance"
)

type ServiceRecord struct {
	ID          string    `json:"id"`
	VendorName  string    `json:"vendor_name"`
	EquipmentID string    `json:"equipment_id"`
	PlanID      string    `json:"plan_id"`
	Description string    `json:"description"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	ServicedAt  time.Time `json:"serviced_at"`
}

var (
	ErrInvalidLedgerEntry    = errors.New("invalid supplier ledger entry")
	ErrPlanEquipmentMismatch = errors.New("supplier service plan belongs to another equipment")
)

type normalizedLedgerEntry struct {
	vendor      string
	equipmentID string
	planID      string
	currency    string
	amount      int64
}

func normalizeLedgerEntry(item ServiceRecord) normalizedLedgerEntry {
	return normalizedLedgerEntry{
		vendor:      strings.TrimSpace(item.VendorName),
		equipmentID: strings.TrimSpace(item.EquipmentID),
		planID:      strings.TrimSpace(item.PlanID),
		currency:    strings.ToUpper(strings.TrimSpace(item.Currency)),
		amount:      item.AmountCents,
	}
}

func validateLedgerIdentity(item normalizedLedgerEntry) error {
	if item.vendor == "" || item.equipmentID == "" {
		return fmt.Errorf("%w: vendor and equipment are required", ErrInvalidLedgerEntry)
	}
	return nil
}

func validateLedgerMoney(item normalizedLedgerEntry) error {
	if item.amount < 0 {
		return fmt.Errorf("%w: amount cannot be negative", ErrInvalidLedgerEntry)
	}
	if item.currency != "CNY" && item.currency != "USD" {
		return fmt.Errorf("%w: unsupported currency", ErrInvalidLedgerEntry)
	}
	return nil
}

func validateReferencedPlan(item normalizedLedgerEntry, linkedPlan *maintenance.Plan) error {
	if item.planID == "" {
		return nil
	}
	if linkedPlan == nil || linkedPlan.ID != item.planID {
		return fmt.Errorf("%w: referenced plan does not exist", ErrInvalidLedgerEntry)
	}
	return nil
}

func ValidateServiceRecord(item ServiceRecord, linkedPlan *maintenance.Plan) error {
	normalized := normalizeLedgerEntry(item)
	if err := validateLedgerIdentity(normalized); err != nil {
		return err
	}
	if err := validateLedgerMoney(normalized); err != nil {
		return err
	}
	return validateReferencedPlan(normalized, linkedPlan)
}
