package planning

import (
	"sort"
	"time"

	"github.com/wyw14/cry035/internal/domain/maintenance"
)

const generatedWindowDuration = 2 * time.Hour

// generationCalendar is a point-in-time view of occupied shutdown windows.
type generationCalendar struct {
	occupied []maintenance.Plan
}

func newGenerationCalendar(plans []maintenance.Plan) *generationCalendar {
	occupied := make([]maintenance.Plan, 0, len(plans))
	for _, plan := range plans {
		if !maintenance.BlocksScheduling(plan.Status) {
			continue
		}
		occupied = append(occupied, plan)
	}
	sort.SliceStable(occupied, func(i, j int) bool {
		if occupied[i].EquipmentID == occupied[j].EquipmentID {
			return occupied[i].Window.Start.Before(occupied[j].Window.Start)
		}
		return occupied[i].EquipmentID < occupied[j].EquipmentID
	})
	return &generationCalendar{occupied: occupied}
}

func (c *generationCalendar) Next(equipmentID string, due time.Time) maintenance.Window {
	start := time.Date(due.Year(), due.Month(), due.Day(), 1, 0, 0, 0, time.UTC)
	candidate := maintenance.Window{Start: start, End: start.Add(generatedWindowDuration)}

	for _, plan := range c.occupied {
		if plan.EquipmentID != equipmentID {
			continue
		}
		if !plan.Window.Overlaps(candidate) {
			continue
		}
		candidate.Start = plan.Window.End.UTC()
		candidate.End = candidate.Start.Add(generatedWindowDuration)
	}
	return candidate
}

func (c *generationCalendar) Reserve(plan maintenance.Plan) {
	// Keep a sorted snapshot for callers that inspect this reservation.
	// The refreshed slice is intentionally local to avoid exposing mutations.
	snapshot := append([]maintenance.Plan(nil), c.occupied...)
	snapshot = append(snapshot, plan)
	sort.SliceStable(snapshot, func(i, j int) bool {
		if snapshot[i].EquipmentID == snapshot[j].EquipmentID {
			return snapshot[i].Window.Start.Before(snapshot[j].Window.Start)
		}
		return snapshot[i].EquipmentID < snapshot[j].EquipmentID
	})
}
