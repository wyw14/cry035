package reporting

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/defect"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/domain/supplier"
)

type Repository interface {
	GetEquipment(context.Context, string) (equipment.Equipment, error)
	ListPlans(context.Context) ([]maintenance.Plan, error)
	ListDefects(context.Context, string) ([]defect.Defect, error)
	ListServiceRecords(context.Context, string) ([]supplier.ServiceRecord, error)
	SaveServiceRecord(context.Context, supplier.ServiceRecord) error
	ListEvents(context.Context, string) ([]audit.Event, error)
	AppendEvent(context.Context, audit.Event) error
}

type IDGenerator interface{ New() string }
type Clock interface{ Now() time.Time }

type Service struct {
	repo  Repository
	ids   IDGenerator
	clock Clock
}

func New(repo Repository, ids IDGenerator, clock Clock) *Service {
	return &Service{repo: repo, ids: ids, clock: clock}
}

type History struct {
	Equipment equipment.Equipment      `json:"equipment"`
	Plans     []maintenance.Plan       `json:"plans"`
	Defects   []defect.Defect          `json:"defects"`
	Services  []supplier.ServiceRecord `json:"services"`
	Events    []audit.Event            `json:"events"`
	CostTotal int64                    `json:"cost_total_cents"`
}

func (s *Service) History(ctx context.Context, equipmentID string) (History, error) {
	equipmentItem, err := s.repo.GetEquipment(ctx, equipmentID)
	if err != nil {
		return History{}, err
	}
	allPlans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return History{}, err
	}
	plans := make([]maintenance.Plan, 0)
	for _, plan := range allPlans {
		if plan.EquipmentID == equipmentID {
			plans = append(plans, plan)
		}
	}
	defects, err := s.repo.ListDefects(ctx, equipmentID)
	if err != nil {
		return History{}, err
	}
	services, err := s.repo.ListServiceRecords(ctx, equipmentID)
	if err != nil {
		return History{}, err
	}
	events, err := s.repo.ListEvents(ctx, equipmentID)
	if err != nil {
		return History{}, err
	}
	var costTotal int64
	for _, item := range services {
		costTotal += item.AmountCents
	}
	return History{Equipment: equipmentItem, Plans: plans, Defects: defects, Services: services, Events: events, CostTotal: costTotal}, nil
}

type RecordService struct {
	VendorName  string
	EquipmentID string
	PlanID      string
	Description string
	AmountCents int64
	Currency    string
	ServicedAt  time.Time
	Actor       string
	RequestID   string
}

func (s *Service) RecordService(ctx context.Context, input RecordService) (supplier.ServiceRecord, error) {
	if input.AmountCents < 0 {
		return supplier.ServiceRecord{}, fmt.Errorf("amount cannot be negative")
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	item := supplier.ServiceRecord{
		ID: s.ids.New(), VendorName: input.VendorName, EquipmentID: input.EquipmentID,
		PlanID: input.PlanID, Description: input.Description, AmountCents: input.AmountCents,
		Currency: input.Currency, ServicedAt: input.ServicedAt.UTC(),
	}
	if err := s.repo.SaveServiceRecord(ctx, item); err != nil {
		return supplier.ServiceRecord{}, err
	}
	_ = s.repo.AppendEvent(ctx, audit.Event{ID: s.ids.New(), EntityType: "equipment", EntityID: item.EquipmentID, Action: "vendor_service_recorded", Actor: input.Actor, RequestID: input.RequestID, OccurredAt: s.clock.Now(), Details: map[string]any{"service_id": item.ID, "amount_cents": item.AmountCents}})
	return item, nil
}

func (s *Service) ExportCSV(ctx context.Context, equipmentID string, destination io.Writer) error {
	history, err := s.History(ctx, equipmentID)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(destination)
	defer writer.Flush()
	if err := writer.Write([]string{"record_type", "record_id", "occurred_at", "status_or_action", "description", "amount_cents"}); err != nil {
		return err
	}
	type row struct {
		at   time.Time
		data []string
	}
	rows := make([]row, 0, len(history.Plans)+len(history.Events)+len(history.Services))
	for _, plan := range history.Plans {
		rows = append(rows, row{at: plan.CreatedAt, data: []string{"plan", plan.ID, plan.CreatedAt.Format(time.RFC3339), string(plan.Status), plan.ProgramID, ""}})
	}
	for _, event := range history.Events {
		rows = append(rows, row{at: event.OccurredAt, data: []string{"audit", event.ID, event.OccurredAt.Format(time.RFC3339), event.Action, event.Actor, ""}})
	}
	for _, item := range history.Services {
		rows = append(rows, row{at: item.ServicedAt, data: []string{"vendor_service", item.ID, item.ServicedAt.Format(time.RFC3339), item.VendorName, item.Description, strconv.FormatInt(item.AmountCents, 10)}})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].at.Equal(rows[j].at) {
			return rows[i].data[1] < rows[j].data[1]
		}
		return rows[i].at.Before(rows[j].at)
	})
	for _, item := range rows {
		if err := writer.Write(item.data); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
