package maintenance

import (
	"errors"
	"time"
)

var ErrInvalidWindow = errors.New("maintenance window end must be after start")

// Window uses half-open semantics: [Start, End). Adjacent windows are allowed.
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func NewWindow(start, end time.Time) (Window, error) {
	if !end.After(start) {
		return Window{}, ErrInvalidWindow
	}
	return Window{Start: start.UTC(), End: end.UTC()}, nil
}

func (w Window) Overlaps(other Window) bool {
	return w.Start.Before(other.End) && other.Start.Before(w.End)
}
