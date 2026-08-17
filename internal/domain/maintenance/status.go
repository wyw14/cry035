package maintenance

import "fmt"

type Status string

const (
	StatusPlanned       Status = "planned"
	StatusInProgress    Status = "in_progress"
	StatusSuspended     Status = "suspended"
	StatusOverdue       Status = "overdue"
	StatusPendingReview Status = "pending_review"
	StatusQualified     Status = "qualified"
	StatusRestricted    Status = "restricted"
	StatusRestored      Status = "restored"
	StatusCancelled     Status = "cancelled"
)

var transitions = map[Status]map[Status]struct{}{
	StatusPlanned: {
		StatusInProgress: {}, StatusSuspended: {}, StatusOverdue: {}, StatusCancelled: {},
	},
	StatusInProgress: {
		StatusSuspended: {}, StatusPendingReview: {}, StatusOverdue: {},
	},
	StatusSuspended: {
		StatusInProgress: {}, StatusOverdue: {}, StatusCancelled: {},
	},
	StatusOverdue: {
		StatusInProgress: {}, StatusPendingReview: {}, StatusCancelled: {},
	},
	StatusPendingReview: {
		StatusQualified: {}, StatusRestricted: {},
	},
	StatusRestricted: {
		StatusPendingReview: {},
	},
	StatusQualified: {
		StatusRestored: {},
	},
}

func CanTransition(from, to Status) bool {
	_, ok := transitions[from][to]
	return ok
}

func ValidateTransition(from, to Status) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("illegal maintenance transition %s -> %s", from, to)
	}
	return nil
}

func BlocksScheduling(status Status) bool {
	return status != StatusCancelled
}
