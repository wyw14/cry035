package postgres

import (
	"context"
	"encoding/json"

	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
)

func (s *Store) ListBuildings(ctx context.Context) ([]equipment.Building, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, address FROM buildings ORDER BY name`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]equipment.Building, 0)
	for rows.Next() {
		var item equipment.Building
		if err := rows.Scan(&item.ID, &item.Name, &item.Address); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListEquipment(ctx context.Context) ([]equipment.Equipment, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, name, building_id, model_id, location, responsible_unit,
		       operating_status, version, created_at, updated_at
		FROM equipment ORDER BY code`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]equipment.Equipment, 0)
	for rows.Next() {
		item, scanErr := scanEquipment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanEquipment(row rowScanner) (equipment.Equipment, error) {
	var item equipment.Equipment
	err := row.Scan(&item.ID, &item.Code, &item.Name, &item.BuildingID, &item.ModelID, &item.Location,
		&item.ResponsibleUnit, &item.Status, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetEquipment(ctx context.Context, id string) (equipment.Equipment, error) {
	item, err := scanEquipment(s.pool.QueryRow(ctx, `
		SELECT id, code, name, building_id, model_id, location, responsible_unit,
		       operating_status, version, created_at, updated_at
		FROM equipment WHERE id=$1`, id))
	return item, translate(err)
}

func (s *Store) GetModel(ctx context.Context, id string) (equipment.Model, error) {
	var item equipment.Model
	err := s.pool.QueryRow(ctx, `SELECT id, manufacturer, name, category FROM equipment_models WHERE id=$1`, id).
		Scan(&item.ID, &item.Manufacturer, &item.Name, &item.Category)
	return item, translate(err)
}

func (s *Store) SaveEquipment(ctx context.Context, item equipment.Equipment) error {
	if err := item.Validate(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO equipment(id, code, name, building_id, model_id, location, responsible_unit,
		                      operating_status, version, created_at, updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name, location=EXCLUDED.location,
		 responsible_unit=EXCLUDED.responsible_unit, operating_status=EXCLUDED.operating_status,
		 version=equipment.version+1, updated_at=EXCLUDED.updated_at`,
		item.ID, item.Code, item.Name, item.BuildingID, item.ModelID, item.Location, item.ResponsibleUnit,
		item.Status, item.Version, item.CreatedAt, item.UpdatedAt)
	return translate(err)
}

func (s *Store) ListPrograms(ctx context.Context) ([]maintenance.Program, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, version, statutory_cycle_days, applicable_categories, checklist, active, updated_at
		FROM maintenance_programs ORDER BY name`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]maintenance.Program, 0)
	for rows.Next() {
		item, scanErr := scanProgram(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanProgram(row rowScanner) (maintenance.Program, error) {
	var item maintenance.Program
	var checklist []byte
	err := row.Scan(&item.ID, &item.Name, &item.Version, &item.StatutoryCycleDays, &item.ApplicableCategories, &checklist, &item.Active, &item.UpdatedAt)
	if err != nil {
		return maintenance.Program{}, err
	}
	if err := json.Unmarshal(checklist, &item.Checklist); err != nil {
		return maintenance.Program{}, err
	}
	return item, nil
}

func (s *Store) GetProgram(ctx context.Context, id string) (maintenance.Program, error) {
	item, err := scanProgram(s.pool.QueryRow(ctx, `
		SELECT id, name, version, statutory_cycle_days, applicable_categories, checklist, active, updated_at
		FROM maintenance_programs WHERE id=$1`, id))
	return item, translate(err)
}

func (s *Store) SaveProgram(ctx context.Context, item maintenance.Program) error {
	checklist, err := json.Marshal(item.Checklist)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO maintenance_programs(id,name,version,statutory_cycle_days,applicable_categories,checklist,active,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name, version=maintenance_programs.version+1,
		 statutory_cycle_days=EXCLUDED.statutory_cycle_days, applicable_categories=EXCLUDED.applicable_categories,
		 checklist=EXCLUDED.checklist, active=EXCLUDED.active, updated_at=EXCLUDED.updated_at`,
		item.ID, item.Name, item.Version, item.StatutoryCycleDays, item.ApplicableCategories, checklist, item.Active, item.UpdatedAt)
	return translate(err)
}
