// Package settings is a togo plugin: a shared, typed config store that any other
// plugin can read, write, and share. Values are JSON, addressed by (scope, key)
// — use scope "global", a plugin name, or a tenant id — and persisted in the
// kernel DB. Use the Go helpers (Get/Set) or the REST API at /api/settings.
package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/togo-framework/togo"
)

// ScopeGlobal is the default scope for app-wide settings.
const ScopeGlobal = "global"

func init() {
	togo.RegisterProviderFunc("settings", togo.PriorityService, func(k *togo.Kernel) error {
		db, err := k.SQL(context.Background())
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(context.Background(),
			`CREATE TABLE IF NOT EXISTS settings (scope text NOT NULL, skey text NOT NULL, value text NOT NULL, PRIMARY KEY (scope, skey))`); err != nil {
			return err
		}
		s := &Store{db: db}
		k.Set("settings", s)
		mount(k.Router, s)
		return nil
	})
}

// Store is the settings service placed in the kernel container under "settings".
type Store struct{ db *sql.DB }

// FromKernel returns the settings store, or nil if the plugin isn't installed.
func FromKernel(k *togo.Kernel) *Store {
	if v, ok := k.Get("settings"); ok {
		if s, ok := v.(*Store); ok {
			return s
		}
	}
	return nil
}

// Set stores any JSON-serialisable value under (scope, key).
func (s *Store) Set(ctx context.Context, scope, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO settings (scope, skey, value) VALUES (?, ?, ?) ON CONFLICT (scope, skey) DO UPDATE SET value = excluded.value`,
		scope, key, string(raw))
	return err
}

// Get unmarshals the value at (scope, key) into dst; ok=false if absent.
func (s *Store) Get(ctx context.Context, scope, key string, dst any) (ok bool, err error) {
	var v string
	err = s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE scope = ? AND skey = ?`, scope, key).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(v), dst)
}

// All returns every setting in a scope as raw JSON values.
func (s *Store) All(ctx context.Context, scope string) (map[string]json.RawMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT skey, value FROM settings WHERE scope = ?`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			return nil, err
		}
		out[key] = json.RawMessage(val)
	}
	return out, rows.Err()
}

// Delete removes a setting.
func (s *Store) Delete(ctx context.Context, scope, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE scope = ? AND skey = ?`, scope, key)
	return err
}

// Get is a typed convenience over Store.Get.
func Get[T any](ctx context.Context, s *Store, scope, key string) (T, bool, error) {
	var v T
	ok, err := s.Get(ctx, scope, key, &v)
	return v, ok, err
}

// Set is a typed convenience over Store.Set.
func Set[T any](ctx context.Context, s *Store, scope, key string, v T) error {
	return s.Set(ctx, scope, key, v)
}

func mount(r chi.Router, s *Store) {
	r.Route("/api/settings", func(r chi.Router) {
		r.Get("/{scope}", func(w http.ResponseWriter, req *http.Request) {
			all, err := s.All(req.Context(), chi.URLParam(req, "scope"))
			writeJSON(w, http.StatusOK, all, err)
		})
		r.Get("/{scope}/{key}", func(w http.ResponseWriter, req *http.Request) {
			var raw json.RawMessage
			ok, err := s.Get(req.Context(), chi.URLParam(req, "scope"), chi.URLParam(req, "key"), &raw)
			if err == nil && !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, raw, err)
		})
		r.Put("/{scope}/{key}", func(w http.ResponseWriter, req *http.Request) {
			var v any
			if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			err := s.Set(req.Context(), chi.URLParam(req, "scope"), chi.URLParam(req, "key"), v)
			writeJSON(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
		})
		r.Delete("/{scope}/{key}", func(w http.ResponseWriter, req *http.Request) {
			err := s.Delete(req.Context(), chi.URLParam(req, "scope"), chi.URLParam(req, "key"))
			writeJSON(w, http.StatusOK, map[string]bool{"ok": err == nil}, err)
		})
	})
}

func writeJSON(w http.ResponseWriter, code int, v any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
