package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/wyw14/cry035/internal/application/catalog"
	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/application/review"
	"github.com/wyw14/cry035/internal/config"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/blobstore"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/platform/idgen"
	"github.com/wyw14/cry035/internal/platform/notifier"
	"github.com/wyw14/cry035/internal/repository"
	"github.com/wyw14/cry035/internal/repository/postgres"
	"github.com/wyw14/cry035/internal/service/alerting"
	"github.com/wyw14/cry035/internal/service/scheduler"
	httptransport "github.com/wyw14/cry035/internal/transport/http"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		client := http.Client{Timeout: 2 * time.Second}
		response, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil || response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	if err := run(logger); err != nil {
		logger.Fatal("server stopped", zap.Error(err))
	}
}

func run(logger *zap.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	var store repository.Store
	var closeStore func()
	var ready func(context.Context) error
	if cfg.DatabaseURL != "" {
		pgStore, openErr := postgres.Open(ctx, cfg.DatabaseURL)
		if openErr != nil {
			return openErr
		}
		store = pgStore
		closeStore = pgStore.Close
		ready = pgStore.Ping
	} else {
		memory := repository.NewMemoryStore()
		seedMemory(memory)
		store = memory
		closeStore = func() {}
		ready = func(context.Context) error { return nil }
	}
	defer closeStore()

	ids := idgen.UUID{}
	clk := clock.Real{}
	cat := catalog.New(store, ids, clk)
	plan := planning.New(store, ids, clk)
	execService := execution.New(store, ids, clk)
	reviewService := review.New(store, ids, clk)
	remediationService := remediation.New(store, ids, clk)
	report := reporting.New(store, ids, clk)
	notifierSink := notifier.NewLocal()
	alerts := alerting.New(store, notifierSink)
	_ = scheduler.New(plan, store, ids, clk) // wired for an external/local tick command
	attachments, err := blobstore.NewLocal(cfg.AttachmentDir, cfg.MaxUploadBytes)
	if err != nil {
		return err
	}
	router := httptransport.NewRouter(httptransport.Dependencies{
		Services:        httptransport.Services{Catalog: cat, Planning: plan, Execution: execService, Review: reviewService, Remediation: remediationService, Reporting: report, Alerting: alerts},
		AttachmentStore: attachments, RequestTimeout: cfg.RequestTimeout, MaxUploadBytes: cfg.MaxUploadBytes,
		CORSOrigin: cfg.CORSOrigin, Ready: ready, Logger: logger,
	})
	server := &http.Server{Addr: cfg.Address, Handler: router, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func seedMemory(store *repository.MemoryStore) {
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	store.Seed(
		[]equipment.Building{{ID: "building-a", Name: "云梯中心 A 座", Address: "高新路 88 号"}, {ID: "building-b", Name: "云梯中心 B 座", Address: "高新路 90 号"}},
		[]equipment.Model{{ID: "model-e1", Manufacturer: "华升设备", Name: "HS-E800", Category: "elevator"}, {ID: "model-f1", Manufacturer: "华升设备", Name: "HS-F2000", Category: "freight_elevator"}, {ID: "model-l1", Manufacturer: "安达机械", Name: "AD-L500", Category: "lifting_device"}},
		[]equipment.Equipment{{ID: "equipment-1", Code: "EL-A-01", Name: "A 座 1 号客梯", BuildingID: "building-a", ModelID: "model-e1", Location: "A 座东厅", ResponsibleUnit: "云梯物业工程部", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}, {ID: "equipment-2", Code: "FE-B-01", Name: "B 座货梯", BuildingID: "building-b", ModelID: "model-f1", Location: "B 座卸货区", ResponsibleUnit: "迅达维保一组", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{ID: "program-monthly", Name: "月度安全保养", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator", "freight_elevator"}, Checklist: []maintenance.ChecklistItem{{ID: "door-lock", Name: "层门门锁联动", Required: true, EvidenceReq: true}, {ID: "brake", Name: "制动器间隙", Required: true, Maximum: floatPtr(0.7)}, {ID: "lamp", Name: "轿厢照明"}}, Active: true, UpdatedAt: now}},
	)
}

func floatPtr(value float64) *float64 { return &value }
