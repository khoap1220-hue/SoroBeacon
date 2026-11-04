package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sorotrail/sorobeacon/internal/notify"
	"github.com/sorotrail/sorobeacon/internal/rules"
	"github.com/sorotrail/sorobeacon/internal/store"
)

type deadLetterMockStore struct {
	emptyStore
	groups   []store.DeadLetterGroup
	letters  []store.DeadLetter
	channels []store.Channel
	alerts   map[int64]store.Alert
	deleted  []int64
}

func (m *deadLetterMockStore) ListDeadLetterGroups(_ context.Context, _ store.DeadLetterFilter) ([]store.DeadLetterGroup, error) {
	return m.groups, nil
}

func (m *deadLetterMockStore) ListDeadLetters(_ context.Context, _ store.DeadLetterFilter) ([]store.DeadLetter, error) {
	return m.letters, nil
}

func (m *deadLetterMockStore) ListChannels(_ context.Context, _ bool) ([]store.Channel, error) {
	return m.channels, nil
}

func (m *deadLetterMockStore) GetDeadLetter(_ context.Context, id int64) (*store.DeadLetter, error) {
	for _, dl := range m.letters {
		if dl.ID == id {
			cp := dl
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *deadLetterMockStore) GetAlert(_ context.Context, id int64) (*store.Alert, error) {
	if a, ok := m.alerts[id]; ok {
		return &a, nil
	}
	return nil, store.ErrNotFound
}

func (m *deadLetterMockStore) GetChannel(_ context.Context, id int64) (*store.Channel, error) {
	for _, ch := range m.channels {
		if ch.ID == id {
			cp := ch
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *deadLetterMockStore) DeleteDeadLetter(_ context.Context, id int64) error {
	m.deleted = append(m.deleted, id)
	return nil
}

func TestDeadLettersPageEmptyAndPopulatedWithoutSecretLeak(t *testing.T) {
	lgr := slog.New(slog.NewTextHandler(os.Stdout, nil))
	st := &deadLetterMockStore{
		channels: []store.Channel{
			{
				ID:     7,
				Name:   "ops-webhook",
				Type:   "webhook",
				Config: json.RawMessage(`{"url":"https://hooks.example.com/TOP_SECRET_WEBHOOK_TOKEN_999"}`),
			},
		},
		letters: []store.DeadLetter{
			{
				ID:           42,
				AlertID:      100,
				ChannelID:    7,
				LastError:    "upstream 502 bad gateway",
				AttemptCount: 3,
				LastStatus:   502,
				CreatedAt:    time.Now().Add(-15 * time.Minute),
			},
		},
	}
	srv, err := New(st, rules.NewRegistry(), notify.DefaultFactory(), lgr)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}
	r := srv.Routes()
	req := httptest.NewRequest(http.MethodGet, "/dead-letters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /dead-letters status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "ops-webhook") {
		t.Errorf("expected channel name ops-webhook in rendered page")
	}
	if strings.Contains(body, "TOP_SECRET_WEBHOOK_TOKEN_999") {
		t.Fatalf("channel secret config leaked into dead-letters HTML response")
	}
}

func TestRedriveDeadLetterSuccessAndUnknownID(t *testing.T) {
	lgr := slog.New(slog.NewTextHandler(os.Stdout, nil))
	factory := notify.NewFactory()
	factory.Register("mock-ok", func(_ json.RawMessage) (notify.Notifier, error) {
		return notify.NotifierFunc(func(_ context.Context, _ store.Alert) error {
			return nil
		}), nil
	})
	factory.Register("mock-fail", func(_ json.RawMessage) (notify.Notifier, error) {
		return notify.NotifierFunc(func(_ context.Context, _ store.Alert) error {
			return errors.New("INTERNAL_SIGNING_SECRET_FAILURE_DETAIL")
		}), nil
	})

	st := &deadLetterMockStore{
		channels: []store.Channel{
			{ID: 1, Name: "ch-ok", Type: "mock-ok", Enabled: true},
			{ID: 2, Name: "ch-fail", Type: "mock-fail", Enabled: true},
		},
		alerts: map[int64]store.Alert{
			10: {ID: 10, MonitorID: 1, RuleID: 1, EventID: "ev-10"},
		},
		letters: []store.DeadLetter{
			{ID: 1, AlertID: 10, ChannelID: 1, AttemptCount: 3},
			{ID: 2, AlertID: 10, ChannelID: 2, AttemptCount: 3},
		},
	}
	srv, err := New(st, rules.NewRegistry(), factory, lgr)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}
	r := srv.Routes()

	// Unknown ID -> 404
	req404 := httptest.NewRequest(http.MethodPost, "/dead-letters/999/redrive", nil)
	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)
	if w404.Code != http.StatusNotFound {
		t.Fatalf("unknown dead letter status = %d, want 404", w404.Code)
	}

	// Failing notifier -> 502 without leaking internal notifier error text
	reqFail := httptest.NewRequest(http.MethodPost, "/dead-letters/2/redrive", nil)
	wFail := httptest.NewRecorder()
	r.ServeHTTP(wFail, reqFail)
	if wFail.Code != http.StatusBadGateway {
		t.Fatalf("failed redrive status = %d, want 502", wFail.Code)
	}
	if strings.Contains(wFail.Body.String(), "INTERNAL_SIGNING_SECRET_FAILURE_DETAIL") {
		t.Fatalf("redrive error response leaked raw notifier error text")
	}
}
