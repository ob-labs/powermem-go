// Package sqlite provides SQLite implementation for user profile storage.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store implements UserProfileStore using SQLite as the backend.
type Store struct {
	// db is the SQLite database connection.
	db *sql.DB

	// tableName is the name of the table storing user profiles.
	tableName string
}

// Config contains configuration for creating a SQLite UserProfileStore.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string

	// TableName is the name of the table to use (default: "user_profiles").
	TableName string
}

// NewStore creates a new SQLite UserProfileStore.
//
// Parameters:
//   - cfg: Configuration containing database path and table name
//
// Returns:
//   - *Store: The store instance
//   - error: Error if database connection or table creation fails
func NewStore(cfg *Config) (*Store, error) {
	if cfg.TableName == "" {
		cfg.TableName = "user_profiles"
	}

	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:        db,
		tableName: cfg.TableName,
	}

	// Create table
	if err := store.initTable(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

// initTable initializes the database table structure.
func (s *Store) initTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			user_id TEXT NOT NULL UNIQUE,
			profile_content TEXT,
			topics TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create index
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_user_id ON %s(user_id)
	`, s.tableName, s.tableName)

	_, err = s.db.ExecContext(ctx, indexQuery)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// SaveProfile saves or updates a user profile.
//
// If a profile for the user already exists, it is updated.
// Otherwise, a new profile is created.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: User identifier
//   - profileContent: Unstructured profile content (optional)
//   - topics: Structured topics (optional)
//
// Returns the profile ID and any error.
func (s *Store) SaveProfile(ctx context.Context, userID string, profileContent *string, topics map[string]interface{}) (int64, error) {
	// Check if profile already exists
	var existingID sql.NullInt64
	checkQuery := fmt.Sprintf("SELECT id FROM %s WHERE user_id = ?", s.tableName)
	err := s.db.QueryRowContext(ctx, checkQuery, userID).Scan(&existingID)

	now := time.Now()
	var topicsJSON []byte
	if topics != nil {
		var err error
		topicsJSON, err = json.Marshal(topics)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal topics: %w", err)
		}
	}

	if err == nil && existingID.Valid {
		// Update existing record
		updateQuery := fmt.Sprintf(`
			UPDATE %s 
			SET profile_content = ?, topics = ?, updated_at = ?
			WHERE user_id = ?
		`, s.tableName)

		_, err = s.db.ExecContext(ctx, updateQuery, profileContent, string(topicsJSON), now, userID)
		if err != nil {
			return 0, fmt.Errorf("failed to update profile: %w", err)
		}
		return existingID.Int64, nil
	} else if err == sql.ErrNoRows {
		// Insert new record
		// Uses simple auto-increment ID (should use Snowflake ID in production)
		insertQuery := fmt.Sprintf(`
			INSERT INTO %s (user_id, profile_content, topics, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, s.tableName)

		result, err := s.db.ExecContext(ctx, insertQuery, userID, profileContent, string(topicsJSON), now, now)
		if err != nil {
			return 0, fmt.Errorf("failed to insert profile: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("failed to get last insert id: %w", err)
		}
		return id, nil
	} else {
		return 0, fmt.Errorf("failed to check existing profile: %w", err)
	}
}

// GetProfileByUserID retrieves a user profile by user ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - userID: User identifier
//
// Returns the UserProfile if found, or nil if not found.
func (s *Store) GetProfileByUserID(ctx context.Context, userID string) (*UserProfile, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, profile_content, topics, created_at, updated_at
		FROM %s
		WHERE user_id = ?
	`, s.tableName)

	var profile UserProfile
	var topicsJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID,
		&profile.UserID,
		&profile.ProfileContent,
		&topicsJSON,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Parse topics JSON
	if topicsJSON.Valid && topicsJSON.String != "" {
		if err := json.Unmarshal([]byte(topicsJSON.String), &profile.Topics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal topics: %w", err)
		}
	}

	return &profile, nil
}

// GetProfiles retrieves a list of user profiles with optional filtering.
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Filtering and pagination options
//
// Returns a list of matching user profiles.
func (s *Store) GetProfiles(ctx context.Context, opts *GetProfilesOptions) ([]*UserProfile, error) {
	query := fmt.Sprintf(`
		SELECT id, user_id, profile_content, topics, created_at, updated_at
		FROM %s
	`, s.tableName)

	args := []interface{}{}
	conditions := []string{}

	if opts.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, opts.UserID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY updated_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var profiles []*UserProfile
	for rows.Next() {
		var profile UserProfile
		var topicsJSON sql.NullString

		err := rows.Scan(
			&profile.ID,
			&profile.UserID,
			&profile.ProfileContent,
			&topicsJSON,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}

		// Parse topics JSON
		if topicsJSON.Valid && topicsJSON.String != "" {
			if err := json.Unmarshal([]byte(topicsJSON.String), &profile.Topics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal topics: %w", err)
			}
		}

		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return profiles, nil
}

// DeleteProfile deletes a user profile by profile ID.
//
// Parameters:
//   - ctx: Context for cancellation
//   - profileID: Profile ID to delete
//
// Returns an error if deletion fails or profile is not found.
func (s *Store) DeleteProfile(ctx context.Context, profileID int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.tableName)
	result, err := s.db.ExecContext(ctx, query, profileID)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("profile not found")
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
