package main

import (
	"backend/db"
	"backend/model"
	"encoding/json"
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

	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		var event model.Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			log.Println("failed to parse request body")
			_, _ = w.Write([]byte(`{"message": "failed to parse request body"}`))
			return
		}

		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
