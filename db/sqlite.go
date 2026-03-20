package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func OpenAndMigrate(fileName string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", fileName)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	const query = `
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			payload TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
		CREATE INDEX IF NOT EXISTS idx_events_type_payload_timestamp ON events(type, payload, timestamp);
	`

	_, err := db.Exec(query)
	return err
}
