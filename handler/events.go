package handler

import (
	"backend/model"
	"backend/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

const apiTimeLayout = time.RFC3339

type EventsHandler struct {
	DB *sql.DB
}

type CreateEventRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type StatsResponse struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func (h *EventsHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid create event request body", "err", err)
		writeError(w, http.StatusBadRequest, "failed to parse request body")
		return
	}

	if strings.TrimSpace(req.Type) == "" {
		log.Warn("create event validation failed", "reason", "missing type")
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	if len(req.Payload) == 0 || !json.Valid(req.Payload) {
		log.Warn("create event validation failed", "reason", "invalid payload")
		writeError(w, http.StatusBadRequest, "payload must be valid json")
		return
	}

	const dedupeQuery = `
		SELECT EXISTS(
			SELECT 1
			FROM events
			WHERE type = ?
				AND payload = ?
				AND timestamp >= datetime('now', '-5 minutes')
			LIMIT 1
		)
	`

	var exists int
	if err := h.DB.QueryRow(dedupeQuery, req.Type, string(req.Payload)).Scan(&exists); err != nil {
		log.Error("failed to check duplicate event", "type", req.Type, "err", err)
		writeError(w, http.StatusConflict, "failed to check duplicate event")
		return
	}

	if exists == 1 {
		log.Info("duplicate event rejected", "type", req.Type)
		writeError(w, http.StatusConflict, "duplicate event within last 5 minutes")
		return
	}

	const insertQuery = `
		INSERT INTO events(type, payload, timestamp)
		VALUES(?, ?, datetime('now'))
	`

	if _, err := h.DB.Exec(insertQuery, req.Type, string(req.Payload)); err != nil {
		log.Error("failed to create event", "type", req.Type, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	log.Info("event created", "type", req.Type)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"created","message":"event created"}`))
}

func (h *EventsHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	where := []string{"1=1"}
	args := []any{}

	eventType := strings.TrimSpace(queryParams.Get("type"))
	if eventType != "" {
		where = append(where, "type = ?")
		args = append(args, eventType)
	}

	startRaw := queryParams.Get("start")
	endRaw := queryParams.Get("end")
	limit := utils.ParseIntWithFallback(queryParams.Get("limit"), 50)
	offset := utils.ParseIntWithFallback(queryParams.Get("offset"), 0)

	start, end, validTimeFilter, err := resolveTimeFilter(startRaw, endRaw)
	if err != nil {
		log.Warn("invalid list events time filter", "from", startRaw, "to", endRaw, "err", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if validTimeFilter {
		where = append(where, "timestamp >= ?", "timestamp <= ?")
		args = append(args,
			start.UTC().Format(time.DateTime),
			end.UTC().Format(time.DateTime),
		)
	}

	query := `
    	SELECT id, type, payload, timestamp
    	FROM events
    	WHERE ` + strings.Join(where, " AND ") + `
    	ORDER BY timestamp DESC
    	LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Error("failed to list events", "type", eventType, "limit", limit, "offset", offset, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	defer rows.Close()

	var events []model.Event

	for rows.Next() {
		var event model.Event
		var payloadText string
		var timestampText string

		if err := rows.Scan(&event.ID, &event.Type, &payloadText, &timestampText); err != nil {
			log.Error("failed to scan event row", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to read event data")
			return
		}

		event.Payload = json.RawMessage(payloadText)
		if !json.Valid(event.Payload) {
			log.Error("invalid json payload stored in database", "event_id", event.ID)
			writeError(w, http.StatusInternalServerError, "stored payload is not valid json")
			return
		}

		ts, err := time.Parse(apiTimeLayout, timestampText)
		if err != nil {
			log.Error("invalid timestamp format stored in database", "event_id", event.ID, "value", timestampText, "err", err)
			writeError(w, http.StatusInternalServerError, "invalid stored timestamp format")
			return
		}
		event.Timestamp = ts.Unix()

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		log.Error("failed while iterating event rows", "err", err)
		writeError(w, http.StatusInternalServerError, "failed while reading events")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"total": len(events),
		"data":  events,
	}); err != nil {
		log.Error("failed to encode list events response", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	log.Info("events listed", "count", len(events), "type", eventType, "limit", limit, "offset", offset, "start", start.String(), "end", end.String())
}

func (h *EventsHandler) EventsStats(w http.ResponseWriter, r *http.Request) {
	args := []any{}
	where := []string{"1=1"}

	queryParams := r.URL.Query()
	startRaw := queryParams.Get("start")
	endRaw := queryParams.Get("end")

	start, end, validTimeFilter, err := resolveTimeFilter(startRaw, endRaw)
	if err != nil {
		log.Warn("invalid events stats time filter", "from", startRaw, "to", endRaw, "err", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if validTimeFilter {
		where = append(where, "timestamp >= ?", "timestamp <= ?")
		args = append(args,
			start.UTC().Format(time.DateTime),
			end.UTC().Format(time.DateTime),
		)
	}

	query := `
		SELECT type, COUNT(*)
		FROM events
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY type
	`

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Error("failed to query events stats", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to query events stats")
		return
	}
	defer rows.Close()

	var stats []StatsResponse

	for rows.Next() {
		var stat StatsResponse
		if err := rows.Scan(&stat.Type, &stat.Count); err != nil {
			log.Error("failed to scan events stats row", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to read events stats data")
			return
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		log.Error("failed while iterating events stats rows", "err", err)
		writeError(w, http.StatusInternalServerError, "failed while reading events stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"total": len(stats),
		"data":  stats,
	}); err != nil {
		log.Error("failed to encode events stats response", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}

	log.Info("events stats retrieved", "count", len(stats), "start", start.String(), "end", end.String())
}

func resolveTimeFilter(startRaw, endRaw string) (time.Time, time.Time, bool, error) {
	if startRaw == "" {
		// ASSUMPTION: If only "end" is provided, time filtering is intentionally ignored.
		return time.Time{}, time.Time{}, false, nil
	}

	start, err := time.Parse(apiTimeLayout, startRaw)
	if err != nil {
		return time.Time{}, time.Time{}, false, errors.New("invalid start date, expected RFC3339")
	}

	var end time.Time
	if endRaw == "" {
		end = time.Now().UTC()
	} else {
		// ASSUMPTION: If only "start" is provided, "end" is set to the current time.
		end, err = time.Parse(apiTimeLayout, endRaw)
		if err != nil {
			return time.Time{}, time.Time{}, false, errors.New("invalid end date, expected RFC3339")
		}
	}

	if !start.Before(end) {
		return time.Time{}, time.Time{}, false, errors.New("invalid time filter: start must be before end")
	}

	return start, end, true, nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"message":"` + msg + `"}`))
}
