package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/inspection"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	baserepo "github.com/wyw14/cry035/internal/repository"
)

func (s *Store) SaveExecution(ctx context.Context, execution inspection.Execution, findings []inspection.Finding, defects []defect.Defect, expectedVersion int64, at time.Time) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	plan, err := scanPlan(tx.QueryRow(ctx, `SELECT `+planColumns+` FROM maintenance_plans WHERE id=$1 FOR UPDATE`, execution.PlanID))
	if err != nil {
		return translate(err)
	}
	if plan.Version != expectedVersion {
		return baserepo.ErrVersionConflict
	}
	if plan.Status != maintenance.StatusInProgress && plan.Status != maintenance.StatusOverdue {
		return baserepo.ErrConflict
	}
	checklist, err := json.Marshal(execution.Checklist)
	if err != nil {
		return err
	}
	measurements, err := json.Marshal(execution.Measurements)
	if err != nil {
		return err
	}
	evidence, err := json.Marshal(execution.Evidence)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO executions(id,plan_id,technician,checklist_snapshot,measurements,evidence,submitted_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, execution.ID, execution.PlanID, execution.Technician,
		checklist, measurements, evidence, execution.SubmittedAt)
	if err != nil {
		return translate(err)
	}
	for _, item := range defects {
		_, err = tx.Exec(ctx, `
			INSERT INTO defects(id,equipment_id,plan_id,execution_id,checklist_item_id,level,description,status,due_at,version,restriction_key)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, item.ID, item.EquipmentID, item.PlanID,
			item.ExecutionID, item.ChecklistItem, item.Level, item.Description, item.Status, item.DueAt, item.Version, item.RestrictionKey)
		if err != nil {
			return translate(err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status='pending_review', version=version+1, updated_at=$2 WHERE id=$1`, plan.ID, at.UTC())
	if err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}

func (s *Store) GetExecutionByPlan(ctx context.Context, planID string) (inspection.Execution, error) {
	var item inspection.Execution
	var checklist, measurements, evidence []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id,plan_id,technician,checklist_snapshot,measurements,evidence,submitted_at,
		       COALESCE(review_comment,''),COALESCE(review_outcome,''),COALESCE(reviewer,''),reviewed_at
		FROM executions WHERE plan_id=$1`, planID).Scan(&item.ID, &item.PlanID, &item.Technician,
		&checklist, &measurements, &evidence, &item.SubmittedAt, &item.ReviewComment, &item.ReviewOutcome,
		&item.Reviewer, &item.ReviewedAt)
	if err != nil {
		return inspection.Execution{}, translate(err)
	}
	if err := json.Unmarshal(checklist, &item.Checklist); err != nil {
		return inspection.Execution{}, err
	}
	if err := json.Unmarshal(measurements, &item.Measurements); err != nil {
		return inspection.Execution{}, err
	}
	if err := json.Unmarshal(evidence, &item.Evidence); err != nil {
		return inspection.Execution{}, err
	}
	return item, nil
}

func (s *Store) ApplyReview(ctx context.Context, planID, executionID, reviewer, comment string, outcome inspection.ReviewOutcome, expectedVersion int64, restrictionKey string, at time.Time) (maintenance.Plan, error) {
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
	if plan.Status != maintenance.StatusPendingReview {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	var storedPlanID string
	if err := tx.QueryRow(ctx, `SELECT plan_id FROM executions WHERE id=$1 FOR UPDATE`, executionID).Scan(&storedPlanID); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if storedPlanID != planID {
		return maintenance.Plan{}, baserepo.ErrConflict
	}
	target := maintenance.StatusQualified
	equipmentStatus := equipment.StatusMaintenance
	if outcome == inspection.ReviewFailed {
		target = maintenance.StatusRestricted
		equipmentStatus = equipment.StatusLimited
		plan.EverRestricted = true
		plan.RestrictionKey = restrictionKey
	}
	if outcome != inspection.ReviewPassed && outcome != inspection.ReviewFailed {
		return maintenance.Plan{}, fmt.Errorf("invalid review outcome")
	}
	_, err = tx.Exec(ctx, `UPDATE executions SET review_comment=$2,review_outcome=$3,reviewer=$4,reviewed_at=$5 WHERE id=$1`, executionID, comment, outcome, reviewer, at.UTC())
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	plan.Status = target
	plan.Version++
	plan.UpdatedAt = at.UTC()
	_, err = tx.Exec(ctx, `UPDATE maintenance_plans SET status=$2,ever_restricted=$3,restriction_key=NULLIF($4,''),version=$5,updated_at=$6 WHERE id=$1`, planID, plan.Status, plan.EverRestricted, plan.RestrictionKey, plan.Version, plan.UpdatedAt)
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	_, err = tx.Exec(ctx, `UPDATE equipment SET operating_status=$2,version=version+1,updated_at=$3 WHERE id=$1`, plan.EquipmentID, equipmentStatus, at.UTC())
	if err != nil {
		return maintenance.Plan{}, translate(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return maintenance.Plan{}, translate(err)
	}
	return plan, nil
}
