package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB      *sql.DB
	DataDir string
	BlobDir string
}

func Open(dbPath, dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "theme-blobs"), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, DataDir: dataDir, BlobDir: filepath.Join(dataDir, "theme-blobs")}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS admins (id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, totp_secret TEXT NOT NULL DEFAULT '', totp_enabled INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, last_login_at INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE, token_hash TEXT NOT NULL UNIQUE, csrf_token TEXT NOT NULL, expires_at INTEGER NOT NULL, created_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS api_tokens (id TEXT PRIMARY KEY, admin_id TEXT NOT NULL REFERENCES admins(id) ON DELETE CASCADE, name TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, created_at INTEGER NOT NULL, revoked_at INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS theme_catalog (id TEXT PRIMARY KEY, package_id TEXT NOT NULL UNIQUE, name TEXT NOT NULL, author TEXT NOT NULL, description TEXT NOT NULL, tags_json TEXT NOT NULL, brightness_support TEXT NOT NULL, category TEXT NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS theme_versions (id TEXT PRIMARY KEY, theme_id TEXT NOT NULL REFERENCES theme_catalog(id) ON DELETE CASCADE, version TEXT NOT NULL, sha256 TEXT NOT NULL, size_bytes INTEGER NOT NULL, expanded_bytes INTEGER NOT NULL, storage_key TEXT NOT NULL, manifest_json TEXT NOT NULL, theme_json TEXT NOT NULL, preview_colors_json TEXT NOT NULL, status TEXT NOT NULL CHECK(status IN ('draft','published','withdrawn','deleted')), created_at INTEGER NOT NULL, published_at INTEGER NOT NULL DEFAULT 0, withdrawn_at INTEGER NOT NULL DEFAULT 0, deleted_at INTEGER NOT NULL DEFAULT 0, UNIQUE(theme_id, version));
CREATE TABLE IF NOT EXISTS audit_events (id INTEGER PRIMARY KEY AUTOINCREMENT, admin_id TEXT NOT NULL, action TEXT NOT NULL, target TEXT NOT NULL, details_json TEXT NOT NULL DEFAULT '{}', created_at INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS idx_versions_public ON theme_versions(theme_id,status);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_events(created_at DESC);`)
	return err
}

func Now() int64                    { return time.Now().UTC().UnixMilli() }
func ID(prefix string) string       { return fmt.Sprintf("%s%x", prefix, time.Now().UnixNano()) }
func JSON(v any) string             { b, _ := json.Marshal(v); return string(b) }
func ParseJSON[T any](raw string) T { var v T; _ = json.Unmarshal([]byte(raw), &v); return v }
func (s *Store) Audit(ctx context.Context, tx *sql.Tx, adminID, action, target string, details any) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(admin_id,action,target,details_json,created_at) VALUES(?,?,?,?,?)`, adminID, action, target, JSON(details), Now())
	return err
}
func (s *Store) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (s *Store) BlobPath(key string) string { return filepath.Join(s.BlobDir, key) }
func (s *Store) RemoveBlobIfUnreferenced(ctx context.Context, key string) {
	var n int
	if s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM theme_versions WHERE storage_key=?`, key).Scan(&n) == nil && n == 0 {
		_ = os.Remove(s.BlobPath(key))
	}
}

var ErrNotFound = errors.New("not found")
