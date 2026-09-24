package settings

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func sqliteStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE settings (scope text NOT NULL, skey text NOT NULL, value text NOT NULL, PRIMARY KEY (scope, skey))`); err != nil {
		t.Fatal(err)
	}
	return &Store{db: db, ph: func(int) string { return "?" }}
}

func TestStoreRoundTrip(t *testing.T) {
	s := sqliteStore(t)
	ctx := context.Background()
	if err := Set(ctx, s, "global", "brand", map[string]string{"name": "Acme"}); err != nil {
		t.Fatal(err)
	}
	if err := Set(ctx, s, "global", "brand", map[string]string{"name": "Acme 2"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, ok, err := Get[map[string]string](ctx, s, "global", "brand")
	if err != nil || !ok || got["name"] != "Acme 2" {
		t.Fatalf("get = %v %v %v", got, ok, err)
	}
	all, err := s.All(ctx, "global")
	if err != nil || len(all) != 1 {
		t.Fatalf("all = %v %v", all, err)
	}
	if err := s.Delete(ctx, "global", "brand"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := Get[map[string]string](ctx, s, "global", "brand"); ok {
		t.Fatal("deleted key still present")
	}
}

// Postgres needs $n placeholders; v0.1.1 hard-coded "?" and failed there.
func TestPlaceholdersFollowTheDialect(t *testing.T) {
	s := &Store{ph: func(n int) string { return "$" + string(rune('0'+n)) }}
	if s.p(1) != "$1" || s.p(3) != "$3" {
		t.Fatalf("got %q %q", s.p(1), s.p(3))
	}
	if (&Store{}).p(2) != "?" {
		t.Fatal("no dialect should fall back to ?")
	}
}

func router(s *Store) http.Handler {
	r := chi.NewRouter()
	mount(r, s)
	return r
}

func request(h http.Handler, method, path, body string, header map[string]string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestHTTPClosedByDefault(t *testing.T) {
	SetAuthorizer(nil)
	t.Setenv("SETTINGS_HTTP_OPEN", "")
	h := router(sqliteStore(t))
	if code := request(h, http.MethodPut, "/api/settings/global/brand", `{"name":"x"}`, nil); code != http.StatusForbidden {
		t.Fatalf("SECURITY: an unauthorised write must be refused, got %d", code)
	}
	if code := request(h, http.MethodGet, "/api/settings/global", "", nil); code != http.StatusForbidden {
		t.Fatalf("SECURITY: an unauthorised read must be refused, got %d", code)
	}
}

func TestAuthorizerDecides(t *testing.T) {
	t.Cleanup(func() { SetAuthorizer(nil) })
	SetAuthorizer(func(r *http.Request, write bool) bool {
		role := r.Header.Get("X-Role")
		return role == "admin" || (!write && role == "viewer")
	})
	h := router(sqliteStore(t))

	if code := request(h, http.MethodPut, "/api/settings/global/brand", `{"name":"x"}`, map[string]string{"X-Role": "viewer"}); code != http.StatusForbidden {
		t.Fatalf("viewer must not write, got %d", code)
	}
	if code := request(h, http.MethodPut, "/api/settings/global/brand", `{"name":"x"}`, map[string]string{"X-Role": "admin"}); code != http.StatusOK {
		t.Fatalf("admin should write, got %d", code)
	}
	if code := request(h, http.MethodGet, "/api/settings/global/brand", "", map[string]string{"X-Role": "viewer"}); code != http.StatusOK {
		t.Fatalf("viewer should read, got %d", code)
	}
	if code := request(h, http.MethodDelete, "/api/settings/global/brand", "", map[string]string{"X-Role": "viewer"}); code != http.StatusForbidden {
		t.Fatalf("viewer must not delete, got %d", code)
	}
}

func TestDevOpenFlagNeverAppliesInProduction(t *testing.T) {
	SetAuthorizer(nil)
	h := router(sqliteStore(t))

	t.Setenv("SETTINGS_HTTP_OPEN", "true")
	t.Setenv("APP_ENV", "development")
	if code := request(h, http.MethodPut, "/api/settings/global/k", `1`, nil); code != http.StatusOK {
		t.Fatalf("dev opt-in should open the API, got %d", code)
	}
	t.Setenv("APP_ENV", "production")
	if code := request(h, http.MethodPut, "/api/settings/global/k", `1`, nil); code != http.StatusForbidden {
		t.Fatalf("SECURITY: SETTINGS_HTTP_OPEN must be ignored in production, got %d", code)
	}
}
