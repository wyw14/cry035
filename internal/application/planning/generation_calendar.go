package planning

import (
	"sort"
	"time"

	"github.com/wyw14/cry035/internal/domain/maintenance"
)

const generatedWindowDuration = 2 * time.Hour

// generationCalendar tracks both stored plans and plans reserved in this batch.
type generationCalendar struct {
	byEquipment map[string][]maintenance.Window
}

func newGenerationCalendar(plans []maintenance.Plan) *generationCalendar {
	calendar := &generationCalendar{byEquipment: make(map[string][]maintenance.Window)}
	for _, plan := range plans {
		if !maintenance.BlocksScheduling(plan.Status) {
			continue
		}
		calendar.byEquipment[plan.EquipmentID] = append(
			calendar.byEquipment[plan.EquipmentID],
			plan.Window,
		)
	}
	for equipmentID := range calendar.byEquipment {
		calendar.sort(equipmentID)
	}
	return calendar
}

func (c *generationCalendar) Next(equipmentID string, due time.Time) maintenance.Window {
	start := time.Date(due.Year(), due.Month(), due.Day(), 1, 0, 0, 0, time.UTC)
	for {
		candidate := maintenance.Window{Start: start, End: start.Add(generatedWindowDuration)}
		conflictEnd, conflict := c.firstConflictEnd(equipmentID, candidate)
		if !conflict {
			return candidate
		}
		start = conflictEnd
	}
}

func (c *generationCalendar) firstConflictEnd(equipmentID string, candidate maintenance.Window) (time.Time, bool) {
	for _, occupied := range c.byEquipment[equipmentID] {
		if occupied.Start.After(candidate.End) || occupied.Start.Equal(candidate.End) {
			break
		}
		if occupied.Overlaps(candidate) {
			return occupied.End.UTC(), true
		}
	}
	return time.Time{}, false
}

func (c *generationCalendar) Reserve(plan maintenance.Plan) {
	if !maintenance.BlocksScheduling(plan.Status) {
		return
	}
	c.byEquipment[plan.EquipmentID] = append(c.byEquipment[plan.EquipmentID], plan.Window)
	c.sort(plan.EquipmentID)
}

func (c *generationCalendar) sort(equipmentID string) {
	windows := c.byEquipment[equipmentID]
	sort.SliceStable(windows, func(i, j int) bool {
		if windows[i].Start.Equal(windows[j].Start) {
			return windows[i].End.Before(windows[j].End)
		}
		return windows[i].Start.Before(windows[j].Start)
	})
	c.byEquipment[equipmentID] = windows
}
