package maintenance

import "testing"

func TestStateMachineRejectsDirectRestrictedRestore(t *testing.T) {
	if CanTransition(StatusRestricted, StatusRestored) {
		t.Fatal("restricted plan must pass reinspection review before restore")
	}
	if !CanTransition(StatusRestricted, StatusPendingReview) {
		t.Fatal("restricted plan should accept a reinspection review")
	}
}
