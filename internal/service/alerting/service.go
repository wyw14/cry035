package alerting

import (
	"context"

	"github.com/wyw14/cry035/internal/domain/audit"
	"github.com/wyw14/cry035/internal/platform/notifier"
)

type Repository interface {
	ListAlerts(context.Context) ([]audit.Alert, error)
}

type Service struct {
	repo Repository
	sink *notifier.Local
}

func New(repo Repository, sink *notifier.Local) *Service {
	return &Service{repo: repo, sink: sink}
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
		if err := s.sink.Send(ctx, notifier.Message{Recipient: recipient, Subject: item.Level + " 安全告警", Body: item.Message, SentAt: item.CreatedAt}); err != nil {
			return err
		}
	}
	return nil
}
