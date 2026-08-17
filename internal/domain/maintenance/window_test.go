package maintenance

import (
	"testing"
	"time"
)

func TestWindowOverlapRules(t *testing.T) {
	base := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	window, err := NewWindow(base, base.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want bool
	}{
		{"same", base, base.Add(2 * time.Hour), true},
		{"contained", base.Add(30 * time.Minute), base.Add(time.Hour), true},
		{"contains", base.Add(-time.Hour), base.Add(3 * time.Hour), true},
		{"adjacent before", base.Add(-time.Hour), base, false},
		{"adjacent after", base.Add(2 * time.Hour), base.Add(3 * time.Hour), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			other, err := NewWindow(tc.from, tc.to)
			if err != nil {
				t.Fatal(err)
			}
			if got := window.Overlaps(other); got != tc.want {
				t.Fatalf("Overlaps() = %v, want %v", got, tc.want)
			}
		})
	}
}
