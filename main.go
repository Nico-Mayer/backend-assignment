package main

import (
	"backend/db"
	"backend/handler"
	"net/http"

	"github.com/charmbracelet/log"
)

func main() {
	log.SetLevel(log.InfoLevel)

	database, err := db.OpenAndMigrate("event-log.db")
	if err != nil {
		log.Error("database initialization failed", "err", err)
		return
	}
	defer database.Close()

	mux := http.NewServeMux()
	eventsHandler := &handler.EventsHandler{DB: database}

	mux.HandleFunc("POST /events", eventsHandler.CreateEvent)
	mux.HandleFunc("GET /events", eventsHandler.ListEvents)
	mux.HandleFunc("GET /events/stats", eventsHandler.EventsStats)

	log.Info("server starting", "addr", "http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Error("server stopped", "err", err)
	}
}
