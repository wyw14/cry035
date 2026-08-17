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

type serviceRecordRule func(ServiceRecord, *maintenance.Plan) error

var ledgerRules = []serviceRecordRule{
	requireLedgerIdentity,
	requireSupportedMoney,
	requirePlanInEquipmentScope,
}

func requireLedgerIdentity(item ServiceRecord, _ *maintenance.Plan) error {
	if strings.TrimSpace(item.VendorName) == "" {
		return fmt.Errorf("%w: vendor is required", ErrInvalidLedgerEntry)
	}
	if strings.TrimSpace(item.EquipmentID) == "" {
		return fmt.Errorf("%w: equipment is required", ErrInvalidLedgerEntry)
	}
	return nil
}

func requireSupportedMoney(item ServiceRecord, _ *maintenance.Plan) error {
	if item.AmountCents < 0 {
		return fmt.Errorf("%w: amount cannot be negative", ErrInvalidLedgerEntry)
	}
	currency := strings.ToUpper(strings.TrimSpace(item.Currency))
	if currency != "CNY" && currency != "USD" {
		return fmt.Errorf("%w: unsupported currency", ErrInvalidLedgerEntry)
	}
	return nil
}

func requirePlanInEquipmentScope(item ServiceRecord, linkedPlan *maintenance.Plan) error {
	if strings.TrimSpace(item.PlanID) == "" {
		return nil
	}
	if linkedPlan == nil || linkedPlan.ID != item.PlanID {
		return fmt.Errorf("%w: referenced plan does not exist", ErrInvalidLedgerEntry)
	}
	if !linkedPlan.AcceptsService(item.EquipmentID) {
		return fmt.Errorf("%w: plan %s", ErrPlanEquipmentMismatch, item.PlanID)
	}
	return nil
}

func ValidateServiceRecord(item ServiceRecord, linkedPlan *maintenance.Plan) error {
	for _, rule := range ledgerRules {
		if err := rule(item, linkedPlan); err != nil {
			return err
		}
	}
	return nil
}
