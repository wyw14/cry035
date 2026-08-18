package alerting

import (
	"context"
	"sync"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/platform/notifier"
)

type Repository interface {
	ListAlerts(context.Context) ([]audit.Alert, error)
}

type Service struct {
	repo      Repository
	sink      *notifier.Local
	mu        sync.Mutex
	delivered map[string]struct{}
}

func New(repo Repository, sink *notifier.Local) *Service {
	return &Service{repo: repo, sink: sink, delivered: make(map[string]struct{})}
}

func (s *Service) Alerts(ctx context.Context) ([]audit.Alert, error) {
	return s.repo.ListAlerts(ctx)
}

func (s *Service) Dispatch(ctx context.Context, recipient string) error {
	alerts, err := s.repo.ListAlerts(ctx)
	if err != nil {
		return err
	}
	for _, item := range alerts {
		if item.Read {
			continue
		}
		if !s.reserveDelivery(equipment.AlertDeliveryKey(item.ID)) {
			continue
		}
		if err := s.sink.Send(ctx, notifier.Message{Recipient: recipient, Subject: item.Level + " 安全告警", Body: item.Message, SentAt: item.CreatedAt}); err != nil {
			s.releaseDelivery(equipment.AlertDeliveryKey(item.ID))
			return err
		}
	}
	return nil
}

func (s *Service) reserveDelivery(alertID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.delivered[alertID]; exists {
		return false
	}
	s.delivered[alertID] = struct{}{}
	return true
}

func (s *Service) releaseDelivery(alertID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.delivered, alertID)
}
