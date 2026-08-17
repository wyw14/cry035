package httptransport

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/wyw14/cry035/internal/application/catalog"
	"github.com/wyw14/cry035/internal/application/execution"
	"github.com/wyw14/cry035/internal/application/planning"
	"github.com/wyw14/cry035/internal/application/remediation"
	"github.com/wyw14/cry035/internal/application/reporting"
	"github.com/wyw14/cry035/internal/application/review"
	"github.com/wyw14/cry035/internal/domain/equipment"
	"github.com/wyw14/cry035/internal/domain/maintenance"
	"github.com/wyw14/cry035/internal/platform/blobstore"
	"github.com/wyw14/cry035/internal/platform/clock"
	"github.com/wyw14/cry035/internal/platform/notifier"
	"github.com/wyw14/cry035/internal/repository"
	"github.com/wyw14/cry035/internal/service/alerting"
)

type testIDs struct{ value atomic.Int64 }

func (i *testIDs) New() string { return fmt.Sprintf("test-%d", i.value.Add(1)) }

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	now := time.Date(2026, 8, 17, 8, 0, 0, 0, time.UTC)
	store := repository.NewMemoryStore()
	store.Seed(
		[]equipment.Building{{ID: "b1", Name: "A"}},
		[]equipment.Model{{ID: "m1", Manufacturer: "M", Name: "E", Category: "elevator"}},
		[]equipment.Equipment{{ID: "e1", Code: "E1", Name: "客梯", BuildingID: "b1", ModelID: "m1", ResponsibleUnit: "物业", Status: equipment.StatusRunning, Version: 1, CreatedAt: now, UpdatedAt: now}},
		[]maintenance.Program{{ID: "p1", Name: "月检", Version: 1, StatutoryCycleDays: 30, ApplicableCategories: []string{"elevator"}, Active: true, UpdatedAt: now}},
	)
	ids := &testIDs{}
	clk := clock.Fixed{Time: now}
	attachments, err := blobstore.NewLocal(t.TempDir(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	return NewRouter(Dependencies{
		Services: Services{
			Catalog: catalog.New(store, ids, clk), Planning: planning.New(store, ids, clk),
			Execution: execution.New(store, ids, clk), Review: review.New(store, ids, clk),
			Remediation: remediation.New(store, ids, clk), Reporting: reporting.New(store, ids, clk),
			Alerting: alerting.New(store, notifier.NewLocal()),
		},
		AttachmentStore: attachments, RequestTimeout: time.Second, MaxUploadBytes: 1024,
		CORSOrigin: "http://localhost:5173", Ready: func(context.Context) error { return nil }, Logger: zap.NewNop(),
	})
}

func TestHTTPPermissionAndValidation(t *testing.T) {
	router := testRouter(t)

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/equipment?page=1&page_size=10", nil))
	if list.Code != http.StatusOK || list.Header().Get("X-Request-ID") == "" {
		t.Fatalf("list status=%d request_id=%q body=%s", list.Code, list.Header().Get("X-Request-ID"), list.Body.String())
	}

	forbiddenRequest := httptest.NewRequest(http.MethodPost, "/api/v1/equipment", bytes.NewBufferString(`{"code":"E2"}`))
	forbiddenRequest.Header.Set("Content-Type", "application/json")
	forbiddenRequest.Header.Set("X-Role", "auditor")
	forbidden := httptest.NewRecorder()
	router.ServeHTTP(forbidden, forbiddenRequest)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("forbidden status=%d body=%s", forbidden.Code, forbidden.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/equipment", bytes.NewBufferString(`{"code":"E2"}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	invalidRequest.Header.Set("X-Role", "planner")
	invalid := httptest.NewRecorder()
	router.ServeHTTP(invalid, invalidRequest)
	if invalid.Code != http.StatusBadRequest || !bytes.Contains(invalid.Body.Bytes(), []byte(`"code":"VALIDATION_ERROR"`)) {
		t.Fatalf("invalid status=%d body=%s", invalid.Code, invalid.Body.String())
	}
}
