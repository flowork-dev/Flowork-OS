// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package floworkdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"flowork-gui/internal/sidecar"

	_ "modernc.org/sqlite"
)

func Path() string {

	return sidecar.FloworkDB()
}

type Store struct {
	mu   sync.Mutex
	db   *sql.DB
	Path string
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir db parent: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, Path: path}
	if err := s.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) ensureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS kv (
			k TEXT PRIMARY KEY,
			v TEXT NOT NULL DEFAULT ''
		) WITHOUT ROWID`,
		`CREATE TABLE IF NOT EXISTS secrets (
			k TEXT PRIMARY KEY,
			v TEXT NOT NULL DEFAULT ''
		) WITHOUT ROWID`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}
	return nil
}

func (s *Store) GetKV(k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOne("kv", k)
}

func (s *Store) SetKV(k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.upsert("kv", k, v)
}

func (s *Store) GetSecret(k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := s.getOne("secrets", k)
	if err != nil || raw == "" {
		return raw, err
	}
	if plaintextSecretKeys[k] {
		return raw, nil
	}
	return secretDec(raw), nil
}

func (s *Store) SetSecret(k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := v
	if !plaintextSecretKeys[k] {
		stored = secretEnc(v)
	}
	return s.upsert("secrets", k, stored)
}

func (s *Store) DeleteSecret(k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM secrets WHERE k=?`, k)
	return err
}

func (s *Store) ListSecretKeys() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT k FROM secrets ORDER BY k`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if serr := rows.Scan(&k); serr != nil {
			return nil, serr
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) AllSecrets() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT k, v FROM secrets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if serr := rows.Scan(&k, &v); serr != nil {
			return nil, serr
		}
		if !plaintextSecretKeys[k] {
			v = secretDec(v)
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (s *Store) getOne(table, k string) (string, error) {
	var v string
	err := s.db.QueryRow("SELECT v FROM "+table+" WHERE k = ?", k).Scan(&v) // scanner:ignore — table = literal hardcoded (kv/secrets), key parameterized (?)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) upsert(table, k, v string) error {
	q := fmt.Sprintf("INSERT INTO %s(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v", table) // scanner:ignore — table = literal hardcoded (kv/secrets), value parameterized (?)
	_, err := s.db.Exec(q, k, v)
	return err
}

var (
	sharedMu  sync.Mutex
	sharedDB  *Store
	sharedErr error
)

func Shared() (*Store, error) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedDB == nil && sharedErr == nil {
		sharedDB, sharedErr = Open(Path())
	}
	return sharedDB, sharedErr
}

func DefaultModelShared() string {
	s, err := Shared()
	if err != nil || s == nil {
		return ""
	}
	m, _ := s.GetKV("llm_default_model")
	return strings.TrimSpace(m)
}
