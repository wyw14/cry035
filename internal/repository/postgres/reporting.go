package postgres

import (
	"context"
	"encoding/json"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/supplier"
)

func (s *Store) SaveServiceRecord(ctx context.Context, item supplier.ServiceRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vendor_services(id,vendor_name,equipment_id,plan_id,description,amount_cents,currency,serviced_at)
		VALUES($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8)`, item.ID, item.VendorName, item.EquipmentID,
		item.PlanID, item.Description, item.AmountCents, item.Currency, item.ServicedAt)
	return translate(err)
}

func (s *Store) ListServiceRecords(ctx context.Context, equipmentID string) ([]supplier.ServiceRecord, error) {
	query := `SELECT id,vendor_name,equipment_id,COALESCE(plan_id,''),description,amount_cents,currency,serviced_at FROM vendor_services`
	args := []any{}
	if equipmentID != "" {
		query += ` WHERE equipment_id=$1`
		args = append(args, equipmentID)
	}
	query += ` ORDER BY serviced_at,id`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]supplier.ServiceRecord, 0)
	for rows.Next() {
		var item supplier.ServiceRecord
		if err := rows.Scan(&item.ID, &item.VendorName, &item.EquipmentID, &item.PlanID, &item.Description, &item.AmountCents, &item.Currency, &item.ServicedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) AppendEvent(ctx context.Context, item audit.Event) error {
	details, err := json.Marshal(item.Details)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO audit_events(id,entity_type,entity_id,action,actor,request_id,details,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, item.ID, item.EntityType, item.EntityID, item.Action, item.Actor, item.RequestID, details, item.OccurredAt)
	return translate(err)
}

func (s *Store) ListEvents(ctx context.Context, entityID string) ([]audit.Event, error) {
	query := `SELECT id,entity_type,entity_id,action,actor,request_id,details,occurred_at FROM audit_events`
	args := []any{}
	if entityID != "" {
		query += ` WHERE entity_id=$1`
		args = append(args, entityID)
	}
	query += ` ORDER BY occurred_at,id`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]audit.Event, 0)
	for rows.Next() {
		var item audit.Event
		var details []byte
		if err := rows.Scan(&item.ID, &item.EntityType, &item.EntityID, &item.Action, &item.Actor, &item.RequestID, &details, &item.OccurredAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(details, &item.Details); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SaveAlert(ctx context.Context, item audit.Alert) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO alerts(id,equipment_id,plan_id,level,message,is_read,created_at) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7) ON CONFLICT(id) DO UPDATE SET is_read=EXCLUDED.is_read`, item.ID, item.EquipmentID, item.PlanID, item.Level, item.Message, item.Read, item.CreatedAt)
	return translate(err)
}

func (s *Store) ListAlerts(ctx context.Context) ([]audit.Alert, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,equipment_id,COALESCE(plan_id,''),level,message,is_read,created_at FROM alerts ORDER BY created_at DESC,id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]audit.Alert, 0)
	for rows.Next() {
		var item audit.Alert
		if err := rows.Scan(&item.ID, &item.EquipmentID, &item.PlanID, &item.Level, &item.Message, &item.Read, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
