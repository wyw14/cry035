package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	baserepo "github.com/wyw14/cry035/internal/repository"
)

func scanPlan(row rowScanner) (maintenance.Plan, error) {
	var item maintenance.Plan
	var spares []byte
	err := row.Scan(
		&item.ID, &item.EquipmentID, &item.ProgramID, &item.ProgramVersion, &item.GenerationKey,
		&item.Window.Start, &item.Window.End, &item.Assignee, &spares, &item.Status,
		&item.EverRestricted, &item.RestrictionKey, &item.Version, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return maintenance.Plan{}, err
	}
	if err := json.Unmarshal(spares, &item.Spares); err != nil {
		return maintenance.Plan{}, err
	}
	return item, nil
}

const planColumns = `id, equipment_id, program_id, program_version, COALESCE(generation_key,''),
 window_start, window_end, assignee, spares, status, ever_restricted,
 COALESCE(restriction_key,''), version, created_at, updated_at`

func (s *Store) CreatePlan(ctx context.Context, candidate maintenance.Plan) (maintenance.Plan, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return maintenance.Plan{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, candidate.EquipmentID); err != nil {
		return maintenance.Plan{}, false, err
	}
	if candidate.GenerationKey != "" {
		existing, queryErr := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE generation_key=$1`, candidate.GenerationKey))
		if queryErr == nil {
			if err := tx.Commit(ctx); err != nil {
				return maintenance.Plan{}, false, err
			}
			return existing, false, nil
		}
		if queryErr != pgx.ErrNoRows {
			return maintenance.Plan{}, false, queryErr
		}
	}
	var conflictID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM maintenance_plans
		WHERE equipment_id=$1 AND status <> 'cancelled'
		  AND tstzrange(window_start, window_end, '[)') && tstzrange($2, $3, '[)')
		LIMIT 1`, candidate.EquipmentID, candidate.Window.Start, candidate.Window.End).Scan(&conflictID)
	if err == nil {
		return maintenance.Plan{}, false, fmt.Errorf("%w: maintenance window overlaps plan %s", baserepo.ErrConflict, conflictID)
	}
	if err != pgx.ErrNoRows {
		return maintenance.Plan{}, false, err
	}
	spares, err := json.Marshal(candidate.Spares)
	if err != nil {
		return maintenance.Plan{}, false, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO maintenance_plans(
		 id,equipment_id,program_id,program_version,generation_key,window_start,window_end,
		 assignee,spares,status,ever_restricted,restriction_key,version,created_at,updated_at)
		VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,$14,$15)`,
		candidate.ID, candidate.EquipmentID, candidate.ProgramID, candidate.ProgramVersion, candidate.GenerationKey,
		candidate.Window.Start, candidate.Window.End, candidate.Assignee, spares, candidate.Status,
		candidate.EverRestricted, candidate.RestrictionKey, candidate.Version, candidate.CreatedAt, candidate.UpdatedAt)
	if err != nil {
		return maintenance.Plan{}, false, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, false, translate(err)
	}
	return candidate, true, nil
}

func (s *Store) ListPlans(ctx context.Context) ([]maintenance.Plan, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+planColumns+` FROM maintenance_plans ORDER BY window_start, id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]maintenance.Plan, 0)
	for rows.Next() {
		item, scanErr := scanPlan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetPlan(ctx context.Context, id string) (maintenance.Plan, error) {
	item, err := scanPlan(s.pool.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1`, id))
	return item, translate(err)
}

func (s *Store) TransitionPlan(ctx context.Context, id string, expectedVersion int64, to maintenance.Status, at time.Time) (maintenance.Plan, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return maintenance.Plan{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if current.Version != expectedVersion {
		return maintenance.Plan{}, baserepo.ErrVersionConflict
	}
	if err := maintenance.ValidateTransition(current.Status, to); err != nil {
		return maintenance.Plan{}, err
	}
	current.Status = to
	current.Version++
	current.UpdatedAt = at.UTC()
	_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status=$2, version=$3, updated_at=$4 WHERE id=$1`, id, current.Status, current.Version, current.UpdatedAt)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	return current, nil
}
