package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
)

type eventRepository struct {
	db *DB
	tx *sql.Tx
}

func (r *eventRepository) exec() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db.conn
}

func (r *eventRepository) Create(ctx context.Context, event *storage.SessionEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO session_events (session_id, event_type, timestamp, data)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.exec().ExecContext(ctx, query,
		event.SessionID,
		event.EventType,
		event.Timestamp,
		string(dataJSON),
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	event.ID = id
	return nil
}

func (r *eventRepository) CreateBatch(ctx context.Context, events []*storage.SessionEvent) error {
	if len(events) == 0 {
		return nil
	}

	var tx *sql.Tx
	var err error

	if r.tx != nil {
		tx = r.tx
	} else {
		tx, err = r.db.conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO session_events (session_id, event_type, timestamp, data)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}

		result, err := stmt.ExecContext(ctx,
			event.SessionID,
			event.EventType,
			event.Timestamp,
			string(dataJSON),
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert id: %w", err)
		}

		event.ID = id
	}

	if r.tx == nil {
		return tx.Commit()
	}
	return nil
}

func (r *eventRepository) GetBySessionID(ctx context.Context, sessionID string, limit int) ([]*storage.SessionEvent, error) {
	query := `
		SELECT id, session_id, event_type, timestamp, data
		FROM session_events
		WHERE session_id = ?
		ORDER BY timestamp DESC
	`

	args := []interface{}{sessionID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.exec().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*storage.SessionEvent
	for rows.Next() {
		event, err := r.scanEventFromRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *eventRepository) GetByFilter(ctx context.Context, filter storage.EventFilter) ([]*storage.SessionEvent, error) {
	query, args := r.buildFilterQuery(filter)

	rows, err := r.exec().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*storage.SessionEvent
	for rows.Next() {
		event, err := r.scanEventFromRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *eventRepository) scanEventFromRows(rows *sql.Rows) (*storage.SessionEvent, error) {
	var event storage.SessionEvent
	var dataJSON string

	err := rows.Scan(
		&event.ID,
		&event.SessionID,
		&event.EventType,
		&event.Timestamp,
		&dataJSON,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
		event.Data = make(map[string]interface{})
	}

	return &event, nil
}

func (r *eventRepository) buildFilterQuery(filter storage.EventFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	query := `
		SELECT id, session_id, event_type, timestamp, data
		FROM session_events
	`

	if filter.SessionID != "" {
		conditions = append(conditions, "session_id = ?")
		args = append(args, filter.SessionID)
	}

	if len(filter.EventTypes) > 0 {
		placeholders := make([]string, len(filter.EventTypes))
		for i, eventType := range filter.EventTypes {
			placeholders[i] = "?"
			args = append(args, eventType)
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ",")))
	}

	if !filter.Since.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.Since)
	}

	if !filter.Until.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.Until)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	return query, args
}
