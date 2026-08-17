package supplier

import "time"

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
