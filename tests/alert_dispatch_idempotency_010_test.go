package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/platform/notifier"
	"github.com/wyw14/cry035/internal/repository"
	"github.com/wyw14/cry035/internal/service/alerting"
)

func TestUnreadAlertDispatchIsIdempotent010(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()
	_ = store.SaveAlert(ctx, audit.Alert{ID: "alert-010", EquipmentID: "equipment-010", Level: "critical", Message: "shutdown window exceeded", CreatedAt: time.Date(2026, 8, 18, 13, 0, 0, 0, time.UTC)})
	sink := notifier.NewLocal()
	service := alerting.New(store, sink)
	const dispatchers = 8
	start := make(chan struct{})
	var wait sync.WaitGroup
	for i := 0; i < dispatchers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			if err := service.Dispatch(ctx, "property-team"); err != nil {
				t.Errorf("Dispatch() error = %v", err)
			}
		}()
	}
	close(start)
	wait.Wait()
	if got := len(sink.Messages()); got != 1 {
		t.Fatalf("same unread alert dispatched %d times concurrently, want 1", got)
	}
}
