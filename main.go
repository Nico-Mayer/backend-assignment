package main

import (
	"backend/db"
	"backend/handler"
	"log"
	"net/http"
)

func main() {
	database, err := db.OpenAndMigrate("event-log.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	mux := http.NewServeMux()
	eventsHandler := &handler.EventsHandler{DB: database}

	mux.HandleFunc("POST /events", eventsHandler.CreateEvent)
	mux.HandleFunc("GET /events", eventsHandler.ListEvents)
	mux.HandleFunc("GET /events/stats", eventsHandler.EventsStats)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
