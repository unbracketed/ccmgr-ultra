package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn     *sql.DB
	dbPath   string
	sessions *sessionRepository
	events   *eventRepository
}

func NewDB(dbPath string) (*DB, error) {
	if err := ensureDir(filepath.Dir(dbPath)); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_journal=WAL&_timeout=5000&_foreign_keys=true", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn:   conn,
		dbPath: dbPath,
	}

	db.sessions = &sessionRepository{db: db}
	db.events = &eventRepository{db: db}

	return db, nil
}

func (db *DB) Sessions() storage.SessionRepository {
	return db.sessions
}

func (db *DB) Events() storage.SessionEventRepository {
	return db.events
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) BeginTx(ctx context.Context) (storage.Transaction, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &transaction{
		tx:       tx,
		sessions: &sessionRepository{tx: tx},
		events:   &eventRepository{tx: tx},
	}, nil
}

type transaction struct {
	tx       *sql.Tx
	sessions *sessionRepository
	events   *eventRepository
}

func (t *transaction) Sessions() storage.SessionRepository {
	return t.sessions
}

func (t *transaction) Events() storage.SessionEventRepository {
	return t.events
}

func (t *transaction) Commit() error {
	return t.tx.Commit()
}

func (t *transaction) Rollback() error {
	return t.tx.Rollback()
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
