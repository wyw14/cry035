package reporting

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/supplier"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/repository"
)

type reportIDs int

func (i *reportIDs) New() string { *i++; return fmt.Sprintf("report-%d", *i) }

func TestEquipmentHistoryExportDoesNotMultiplyChildren(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "b1", Name: "A"}},
		[]equipment.Model{{ID: "m1", Manufacturer: "M", Name: "E", Category: "elevator"}},
		[]equipment.Equipment{{ID: "e1", Code: "E1", Name: "客梯", BuildingID: "b1", ModelID: "m1", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}}, nil,
	)
	for index, amount := range []int64{12000, 8000, 5000} {
		err := store.SaveServiceRecord(ctx, supplier.ServiceRecord{ID: fmt.Sprintf("service-%d", index), VendorName: "维保商", EquipmentID: "e1", Description: "服务", AmountCents: amount, Currency: "CNY", ServicedAt: now.Add(time.Duration(index) * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 2; index++ {
		if err := store.AppendEvent(ctx, audit.Event{ID: fmt.Sprintf("event-%d", index), EntityType: "equipment", EntityID: "e1", Action: "evidence_added", Actor: "tech", Details: map[string]any{"photo": index}, OccurredAt: now.Add(time.Duration(index) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	ids := reportIDs(0)
	service := New(store, &ids, clock.Fixed{Time: now})
	history, err := service.History(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if history.CostTotal != 25000 {
		t.Fatalf("cost=%d, want 25000", history.CostTotal)
	}
	if len(history.Events) != 2 || len(history.Services) != 3 {
		t.Fatalf("events=%d services=%d", len(history.Events), len(history.Services))
	}
	var output bytes.Buffer
	if err := service.ExportCSV(ctx, "e1", &output); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1+2+3 {
		t.Fatalf("CSV rows=%d, want 6 including header\n%s", len(rows), output.String())
	}
}
