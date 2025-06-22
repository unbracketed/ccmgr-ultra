package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bcdekker/ccmgr-ultra/internal/storage"
)

type sessionRepository struct {
	db *DB
	tx *sql.Tx
}

func (r *sessionRepository) exec() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if r.tx != nil {
		return r.tx
	}
	return r.db.conn
}

func (r *sessionRepository) Create(ctx context.Context, session *storage.Session) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO sessions (id, name, project, worktree, branch, directory, created_at, updated_at, last_access, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err = r.exec().ExecContext(ctx, query,
		session.ID,
		session.Name,
		session.Project,
		session.Worktree,
		session.Branch,
		session.Directory,
		session.CreatedAt,
		session.UpdatedAt,
		session.LastAccess,
		string(metadataJSON),
	)
	
	return err
}

func (r *sessionRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)

	for key, value := range updates {
		if key == "metadata" {
			if metadataMap, ok := value.(map[string]interface{}); ok {
				metadataJSON, err := json.Marshal(metadataMap)
				if err != nil {
					return fmt.Errorf("failed to marshal metadata: %w", err)
				}
				setParts = append(setParts, "metadata = ?")
				args = append(args, string(metadataJSON))
			}
		} else {
			setParts = append(setParts, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE sessions SET %s WHERE id = ?", strings.Join(setParts, ", "))
	
	_, err := r.exec().ExecContext(ctx, query, args...)
	return err
}

func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := r.exec().ExecContext(ctx, query, id)
	return err
}

func (r *sessionRepository) GetByID(ctx context.Context, id string) (*storage.Session, error) {
	query := `
		SELECT id, name, project, worktree, branch, directory, created_at, updated_at, last_access, metadata
		FROM sessions WHERE id = ?
	`
	
	return r.scanSession(r.exec().QueryRowContext(ctx, query, id))
}

func (r *sessionRepository) GetByName(ctx context.Context, name string) (*storage.Session, error) {
	query := `
		SELECT id, name, project, worktree, branch, directory, created_at, updated_at, last_access, metadata
		FROM sessions WHERE name = ?
	`
	
	return r.scanSession(r.exec().QueryRowContext(ctx, query, name))
}

func (r *sessionRepository) List(ctx context.Context, filter storage.SessionFilter) ([]*storage.Session, error) {
	query, args := r.buildListQuery(filter)
	
	rows, err := r.exec().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*storage.Session
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (r *sessionRepository) Search(ctx context.Context, query string, filter storage.SessionFilter) ([]*storage.Session, error) {
	searchQuery, args := r.buildSearchQuery(query, filter)
	
	rows, err := r.exec().QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*storage.Session
	for rows.Next() {
		session, err := r.scanSessionFromRows(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func (r *sessionRepository) Count(ctx context.Context, filter storage.SessionFilter) (int64, error) {
	query, args := r.buildCountQuery(filter)
	
	var count int64
	err := r.exec().QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *sessionRepository) scanSession(row *sql.Row) (*storage.Session, error) {
	var session storage.Session
	var metadataJSON string
	var project, worktree, branch, directory sql.NullString

	err := row.Scan(
		&session.ID,
		&session.Name,
		&project,
		&worktree,
		&branch,
		&directory,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastAccess,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	session.Project = project.String
	session.Worktree = worktree.String
	session.Branch = branch.String
	session.Directory = directory.String

	if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
		session.Metadata = make(map[string]interface{})
	}

	return &session, nil
}

func (r *sessionRepository) scanSessionFromRows(rows *sql.Rows) (*storage.Session, error) {
	var session storage.Session
	var metadataJSON string
	var project, worktree, branch, directory sql.NullString

	err := rows.Scan(
		&session.ID,
		&session.Name,
		&project,
		&worktree,
		&branch,
		&directory,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.LastAccess,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	session.Project = project.String
	session.Worktree = worktree.String
	session.Branch = branch.String
	session.Directory = directory.String

	if err := json.Unmarshal([]byte(metadataJSON), &session.Metadata); err != nil {
		session.Metadata = make(map[string]interface{})
	}

	return &session, nil
}

func (r *sessionRepository) buildListQuery(filter storage.SessionFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	query := `
		SELECT id, name, project, worktree, branch, directory, created_at, updated_at, last_access, metadata
		FROM sessions
	`

	if filter.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, filter.Project)
	}

	if filter.Worktree != "" {
		conditions = append(conditions, "worktree = ?")
		args = append(args, filter.Worktree)
	}

	if filter.Branch != "" {
		conditions = append(conditions, "branch = ?")
		args = append(args, filter.Branch)
	}

	if !filter.Since.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.Since)
	}

	if !filter.Until.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.Until)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = filter.SortOrder
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

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

func (r *sessionRepository) buildSearchQuery(searchQuery string, filter storage.SessionFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	query := `
		SELECT id, name, project, worktree, branch, directory, created_at, updated_at, last_access, metadata
		FROM sessions
	`

	if searchQuery != "" {
		conditions = append(conditions, "(name LIKE ? OR project LIKE ? OR worktree LIKE ? OR branch LIKE ?)")
		searchPattern := "%" + searchQuery + "%"
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if filter.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, filter.Project)
	}

	if filter.Worktree != "" {
		conditions = append(conditions, "worktree = ?")
		args = append(args, filter.Worktree)
	}

	if filter.Branch != "" {
		conditions = append(conditions, "branch = ?")
		args = append(args, filter.Branch)
	}

	if !filter.Since.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.Since)
	}

	if !filter.Until.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.Until)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder != "" {
		sortOrder = filter.SortOrder
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

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

func (r *sessionRepository) buildCountQuery(filter storage.SessionFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	query := "SELECT COUNT(*) FROM sessions"

	if filter.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, filter.Project)
	}

	if filter.Worktree != "" {
		conditions = append(conditions, "worktree = ?")
		args = append(args, filter.Worktree)
	}

	if filter.Branch != "" {
		conditions = append(conditions, "branch = ?")
		args = append(args, filter.Branch)
	}

	if !filter.Since.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.Since)
	}

	if !filter.Until.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.Until)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return query, args
}