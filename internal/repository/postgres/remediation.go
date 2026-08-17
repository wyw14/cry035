package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	baserepo "github.com/wyw14/cry035/internal/repository"
)

func scanDefect(row rowScanner) (defect.Defect, error) {
	var item defect.Defect
	err := row.Scan(&item.ID, &item.EquipmentID, &item.PlanID, &item.ExecutionID, &item.ChecklistItem,
		&item.Level, &item.Description, &item.Status, &item.DueAt, &item.RectifiedAt, &item.ClosedAt,
		&item.Version, &item.RestrictionKey)
	return item, err
}

const defectColumns = `id,equipment_id,plan_id,execution_id,checklist_item_id,level,description,status,
 due_at,rectified_at,closed_at,version,restriction_key`

func (s *Store) ListDefects(ctx context.Context, equipmentID string) ([]defect.Defect, error) {
	query := `SELECT ` + defectColumns + ` FROM defects`
	args := []any{}
	if equipmentID != "" {
		query += ` WHERE equipment_id=$1`
		args = append(args, equipmentID)
	}
	query += ` ORDER BY due_at,id`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	items := make([]defect.Defect, 0)
	for rows.Next() {
		item, scanErr := scanDefect(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SaveRectification(ctx context.Context, item defect.Rectification, expectedVersion int64, at time.Time) (defect.Defect, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return defect.Defect{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanDefect(tx.QueryRow(ctx, `SELECT `+defectColumns+` FROM defects WHERE id=$1 FOR UPDATE`, item.DefectID))
	if err != nil {
		return defect.Defect{}, translate(err)
	}
	if current.Version != expectedVersion {
		return defect.Defect{}, baserepo.ErrVersionConflict
	}
	if current.Status == defect.StatusClosed {
		return defect.Defect{}, baserepo.ErrConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO rectifications(id,defect_id,action,operator,evidence_id,created_at) VALUES($1,$2,$3,$4,NULLIF($5,''),$6)`, item.ID, item.DefectID, item.Action, item.Operator, item.EvidenceID, item.CreatedAt)
	if err != nil {
		return defect.Defect{}, translate(err)
	}
	current.Status = defect.StatusReadyForCheck
	current.Version++
	rectifiedAt := at.UTC()
	current.RectifiedAt = &rectifiedAt
	_, err = tx.Exec(ctx, `UPDATE defects SET status=$2,version=$3,rectified_at=$4 WHERE id=$1`, current.ID, current.Status, current.Version, current.RectifiedAt)
	if err != nil {
		return defect.Defect{}, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return defect.Defect{}, translate(err)
	}
	return current, nil
}

func (s *Store) SaveReinspection(ctx context.Context, item defect.Reinspection, at time.Time) (maintenance.Plan, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return maintenance.Plan{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	plan, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1 FOR UPDATE`, item.PlanID))
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if plan.EquipmentID != item.EquipmentID || plan.Status != maintenance.StatusRestricted || plan.RestrictionKey != item.RestrictionKey {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	for _, id := range item.DefectIDs {
		current, scanErr := scanDefect(tx.QueryRow(ctx, `SELECT `+defectColumns+` FROM defects WHERE id=$1 FOR UPDATE`, id))
		if scanErr != nil {
			return maintenance.Plan{}, translate(scanErr)
		}
		if current.RestrictionKey != item.RestrictionKey || current.Status != defect.StatusReadyForCheck {
			return maintenance.Plan{}, baserepo.ErrConflict
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO reinspections(id,plan_id,equipment_id,restriction_key,defect_ids,inspector,passed,comment,inspected_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, item.ID, item.PlanID, item.EquipmentID, item.RestrictionKey, item.DefectIDs, item.Inspector, item.Passed, item.Comment, item.InspectedAt)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if item.Passed {
		_, err = tx.Exec(ctx, `UPDATE defects SET status='closed',version=version+1,closed_at=$2 WHERE id=ANY($1)`, item.DefectIDs, at.UTC())
		if err != nil {
			return maintenance.Plan{}, translate(err)
		}
		plan.Status = maintenance.StatusPendingReview
		plan.Version++
		plan.UpdatedAt = at.UTC()
		_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status=$2,version=$3,updated_at=$4 WHERE id=$1`, plan.ID, plan.Status, plan.Version, plan.UpdatedAt)
		if err != nil {
			return maintenance.Plan{}, translate(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	return plan, nil
}

func (s *Store) MarkReinspectionReviewed(ctx context.Context, reinspectionID string, reviewer string, at time.Time) (maintenance.Plan, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return maintenance.Plan{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var planID string
	var passed bool
	err = tx.QueryRow(ctx, `SELECT plan_id,passed FROM reinspections WHERE id=$1 FOR UPDATE`, reinspectionID).Scan(&planID, &passed)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if !passed {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	plan, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1 FOR UPDATE`, planID))
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if plan.Status != maintenance.StatusPendingReview {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	plan.Status = maintenance.StatusQualified
	plan.Version++
	plan.UpdatedAt = at.UTC()
	_, err = tx.Exec(ctx, `UPDATE reinspections SET reviewed_at=$2,reviewer=$3 WHERE id=$1`, reinspectionID, at.UTC(), reviewer)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status=$2,version=$3,updated_at=$4 WHERE id=$1`, plan.ID, plan.Status, plan.Version, plan.UpdatedAt)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	return plan, nil
}

func (s *Store) RestoreEquipment(ctx context.Context, planID string, expectedVersion int64, actor, requestID string, at time.Time) (maintenance.Plan, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return maintenance.Plan{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	plan, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1 FOR UPDATE`, planID))
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if plan.Version != expectedVersion {
		return maintenance.Plan{}, baserepo.ErrVersionConflict
	}
	if plan.Status != maintenance.StatusQualified {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	if plan.EverRestricted {
		var reviewedPassed bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reinspections WHERE plan_id=$1 AND restriction_key=$2 AND passed=true AND reviewed_at IS NOT NULL)`, plan.ID, plan.RestrictionKey).Scan(&reviewedPassed)
		if err != nil || !reviewedPassed {
			return maintenance.Plan{}, baserepo.ErrConflict
		}
		var blocking bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM defects WHERE restriction_key=$1 AND level IN ('critical','major') AND status <> 'closed')`, plan.RestrictionKey).Scan(&blocking)
		if err != nil || blocking {
			return maintenance.Plan{}, baserepo.ErrConflict
		}
	}
	plan.Status = maintenance.StatusRestored
	plan.Version++
	plan.UpdatedAt = at.UTC()
	_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status=$2,version=$3,updated_at=$4 WHERE id=$1`, plan.ID, plan.Status, plan.Version, plan.UpdatedAt)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	_, err = tx.Exec(ctx, `UPDATE equipment SET operating_status=$2,version=version+1,updated_at=$3 WHERE id=$1`, plan.EquipmentID, equipment.StatusRunning, at.UTC())
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(id,entity_type,entity_id,action,actor,request_id,details,occurred_at) VALUES(gen_random_uuid()::text,'equipment',$1,'restored',$2,$3,jsonb_build_object('plan_id',$4,'restriction_key',$5),$6)`, plan.EquipmentID, actor, requestID, plan.ID, plan.RestrictionKey, at.UTC())
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	return plan, nil
}
