package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

var DB *sql.DB

func InitDB(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir db dir: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := conn.Exec(string(schema)); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	DB = conn
	log.Println("[DB] SQLite initialized, schema applied")
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func Exec(query string, args ...any) (sql.Result, error) {
	return DB.Exec(query, args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return DB.QueryRow(query, args...)
}

func Query(query string, args ...any) (*sql.Rows, error) {
	return DB.Query(query, args...)
}
