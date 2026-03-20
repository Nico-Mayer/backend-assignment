package handler_test

import (
	"backend/db"
	"backend/handler"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type listEventsResponse struct {
	Total int `json:"total"`
	Data  []struct {
		ID        int             `json:"id"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		Timestamp int64           `json:"timestamp"`
	} `json:"data"`
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "events-test.db")
	database, err := db.OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	mux := http.NewServeMux()
	eventsHandler := &handler.EventsHandler{DB: database}
	mux.HandleFunc("POST /events", eventsHandler.CreateEvent)
	mux.HandleFunc("GET /events", eventsHandler.ListEvents)
	mux.HandleFunc("GET /events/stats", eventsHandler.EventsStats)

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		_ = database.Close()
	})

	return srv
}

func postEvent(t *testing.T, baseURL string, body string) *http.Response {
	t.Helper()

	resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("failed to POST /events: %v", err)
	}

	return resp
}

func TestCreateAndListEvents_HappyPath(t *testing.T) {
	srv := newTestServer(t)

	createBody := `{"type":"user.created","payload":{"id":1,"email":"a@example.com"}}`
	createResp := postEvent(t, srv.URL, createBody)
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d", http.StatusCreated, createResp.StatusCode)
	}

	listResp, err := http.Get(srv.URL + "/events")
	if err != nil {
		t.Fatalf("failed to GET /events: %v", err)
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list status %d, got %d", http.StatusOK, listResp.StatusCode)
	}

	var got listEventsResponse
	if err := json.NewDecoder(listResp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}

	if got.Total != 1 {
		t.Fatalf("expected total 1, got %d", got.Total)
	}

	if len(got.Data) != 1 {
		t.Fatalf("expected one event in data, got %d", len(got.Data))
	}

	event := got.Data[0]
	if event.Type != "user.created" {
		t.Fatalf("expected event type user.created, got %s", event.Type)
	}

	if !json.Valid(event.Payload) {
		t.Fatalf("expected valid json payload, got %s", string(event.Payload))
	}

	if event.Timestamp <= 0 {
		t.Fatalf("expected positive unix timestamp, got %d", event.Timestamp)
	}
}

func TestCreateEvent_DeduplicatesWithinFiveMinutes(t *testing.T) {
	srv := newTestServer(t)

	body := `{"type":"invoice.sent","payload":{"invoiceId":"inv-1"}}`

	firstResp := postEvent(t, srv.URL, body)
	defer firstResp.Body.Close()
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected first create status %d, got %d", http.StatusCreated, firstResp.StatusCode)
	}

	secondResp := postEvent(t, srv.URL, body)
	defer secondResp.Body.Close()
	if secondResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected second create status %d, got %d", http.StatusConflict, secondResp.StatusCode)
	}
}
