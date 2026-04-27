package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aleksclark/spec-torture/internal/schema"
	_ "modernc.org/sqlite"
)

// Store manages persistence of specs, test cases, and results in SQLite.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting journal mode: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating database: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS specs (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			description TEXT,
			source_url TEXT,
			transport TEXT NOT NULL,
			raw_yaml TEXT,
			loaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS test_cases (
			id TEXT NOT NULL,
			spec_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			severity TEXT NOT NULL DEFAULT 'required',
			category TEXT,
			timeout_ms INTEGER,
			tags TEXT,
			steps_json TEXT,
			setup_json TEXT,
			PRIMARY KEY (spec_id, id),
			FOREIGN KEY (spec_id) REFERENCES specs(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS test_runs (
			id TEXT PRIMARY KEY,
			spec_id TEXT NOT NULL,
			runtime TEXT NOT NULL,
			runtime_version TEXT,
			started_at DATETIME NOT NULL,
			completed_at DATETIME,
			summary_json TEXT,
			FOREIGN KEY (spec_id) REFERENCES specs(id)
		)`,
		`CREATE TABLE IF NOT EXISTS test_results (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			spec_id TEXT NOT NULL,
			test_case_id TEXT NOT NULL,
			runtime TEXT NOT NULL,
			runtime_version TEXT,
			status TEXT NOT NULL,
			duration_ms INTEGER,
			started_at DATETIME,
			completed_at DATETIME,
			steps_json TEXT,
			error_message TEXT,
			raw_log TEXT,
			FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE
		)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

// SaveSpec persists a Spec and its TestCases to the database.
func (s *Store) SaveSpec(spec *schema.Spec, rawYAML string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO specs (id, name, version, description, source_url, transport, raw_yaml)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		spec.ID, spec.Name, spec.Version, spec.Description, spec.SourceURL, spec.Transport, rawYAML,
	)
	if err != nil {
		return fmt.Errorf("saving spec: %w", err)
	}

	_, err = tx.Exec("DELETE FROM test_cases WHERE spec_id = ?", spec.ID)
	if err != nil {
		return fmt.Errorf("clearing old test cases: %w", err)
	}

	for _, tc := range spec.TestCases {
		stepsJSON, _ := json.Marshal(tc.Steps)
		setupJSON, _ := json.Marshal(tc.Setup)
		tagsJSON, _ := json.Marshal(tc.Tags)

		_, err = tx.Exec(
			`INSERT INTO test_cases (id, spec_id, name, description, severity, category, timeout_ms, tags, steps_json, setup_json)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tc.ID, spec.ID, tc.Name, tc.Description, tc.Severity, tc.Category,
			tc.Timeout.Milliseconds(), string(tagsJSON), string(stepsJSON), string(setupJSON),
		)
		if err != nil {
			return fmt.Errorf("saving test case %s: %w", tc.ID, err)
		}
	}

	return tx.Commit()
}

// ListSpecs returns all stored specs (without test cases).
func (s *Store) ListSpecs() ([]schema.Spec, error) {
	rows, err := s.db.Query("SELECT id, name, version, description, source_url, transport FROM specs ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []schema.Spec
	for rows.Next() {
		var sp schema.Spec
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.Version, &sp.Description, &sp.SourceURL, &sp.Transport); err != nil {
			return nil, err
		}
		specs = append(specs, sp)
	}
	return specs, rows.Err()
}

// SaveTestRun persists a TestRun and its results to the database.
func (s *Store) SaveTestRun(run *schema.TestRun) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	summaryJSON, _ := json.Marshal(run.Summary)

	_, err = tx.Exec(
		`INSERT INTO test_runs (id, spec_id, runtime, runtime_version, started_at, completed_at, summary_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.SpecID, run.Runtime, run.RuntimeVersion,
		run.StartedAt, run.CompletedAt, string(summaryJSON),
	)
	if err != nil {
		return fmt.Errorf("saving test run: %w", err)
	}

	for _, r := range run.Results {
		stepsJSON, _ := json.Marshal(r.Steps)

		_, err = tx.Exec(
			`INSERT INTO test_results (id, run_id, spec_id, test_case_id, runtime, runtime_version, status, duration_ms, started_at, completed_at, steps_json, error_message, raw_log)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.ID, run.ID, r.SpecID, r.TestCaseID, r.Runtime, r.RuntimeVersion,
			r.Status, r.Duration.Milliseconds(), r.StartedAt, r.CompletedAt,
			string(stepsJSON), r.ErrorMessage, r.RawLog,
		)
		if err != nil {
			return fmt.Errorf("saving test result %s: %w", r.ID, err)
		}
	}

	return tx.Commit()
}

// GetTestRun retrieves a TestRun by ID, including all results.
func (s *Store) GetTestRun(id string) (*schema.TestRun, error) {
	run := &schema.TestRun{}
	var summaryJSON string
	var completedAt sql.NullTime

	err := s.db.QueryRow(
		"SELECT id, spec_id, runtime, runtime_version, started_at, completed_at, summary_json FROM test_runs WHERE id = ?",
		id,
	).Scan(&run.ID, &run.SpecID, &run.Runtime, &run.RuntimeVersion, &run.StartedAt, &completedAt, &summaryJSON)
	if err != nil {
		return nil, fmt.Errorf("getting test run: %w", err)
	}
	if completedAt.Valid {
		run.CompletedAt = completedAt.Time
	}
	_ = json.Unmarshal([]byte(summaryJSON), &run.Summary)

	rows, err := s.db.Query(
		"SELECT id, spec_id, test_case_id, runtime, runtime_version, status, duration_ms, started_at, completed_at, steps_json, error_message, raw_log FROM test_results WHERE run_id = ? ORDER BY started_at",
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r schema.TestResult
		var durationMs int64
		var stepsJSON string
		var startedAt, completedAt sql.NullTime
		var errorMsg, rawLog sql.NullString

		if err := rows.Scan(
			&r.ID, &r.SpecID, &r.TestCaseID, &r.Runtime, &r.RuntimeVersion,
			&r.Status, &durationMs, &startedAt, &completedAt,
			&stepsJSON, &errorMsg, &rawLog,
		); err != nil {
			return nil, err
		}

		r.Duration = time.Duration(durationMs) * time.Millisecond
		if startedAt.Valid {
			r.StartedAt = startedAt.Time
		}
		if completedAt.Valid {
			r.CompletedAt = completedAt.Time
		}
		if errorMsg.Valid {
			r.ErrorMessage = errorMsg.String
		}
		if rawLog.Valid {
			r.RawLog = rawLog.String
		}
		_ = json.Unmarshal([]byte(stepsJSON), &r.Steps)

		run.Results = append(run.Results, r)
	}

	return run, rows.Err()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
