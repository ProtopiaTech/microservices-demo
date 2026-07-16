package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// newSecurityTxtTestRouter builds a minimal router with only the
// security.txt route registered, without running main() or dialing any
// gRPC backend.
func newSecurityTxtTestRouter() *mux.Router {
	r := mux.NewRouter()
	registerSecurityTxtRoute(r, "")
	return r
}

// splitLines normalizes CRLF/LF line endings and returns non-blank lines.
func splitLines(body string) []string {
	rawLines := strings.Split(body, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" {
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

// fieldLines returns the non-comment lines (lines not starting with '#').
func fieldLines(body string) []string {
	var fields []string
	for _, l := range splitLines(body) {
		if strings.HasPrefix(l, "#") {
			continue
		}
		fields = append(fields, l)
	}
	return fields
}

func TestSecurityTxt_T001_ReturnsOK(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestSecurityTxt_T002_ContentTypeIsTextPlainUTF8(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	got := rec.Header().Get("Content-Type")
	want := "text/plain; charset=utf-8"
	if got != want {
		t.Fatalf("expected Content-Type %q, got %q", want, got)
	}
}

func TestSecurityTxt_T003_BodyContainsContactLine(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	found := false
	for _, l := range splitLines(body) {
		if l == "Contact: mailto:security@example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected body to contain line %q, got body:\n%s", "Contact: mailto:security@example.com", body)
	}
}

func TestSecurityTxt_T004_ExpiresIsFutureRFC3339WithinOneYear(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	now := time.Now()
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	var expiresValue string
	found := false
	for _, l := range splitLines(body) {
		if strings.HasPrefix(l, "Expires:") {
			expiresValue = strings.TrimSpace(strings.TrimPrefix(l, "Expires:"))
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected body to contain an Expires: line, got body:\n%s", body)
	}

	parsed, err := time.Parse(time.RFC3339, expiresValue)
	if err != nil {
		t.Fatalf("expected Expires value %q to parse as RFC3339, got error: %v", expiresValue, err)
	}

	if !parsed.After(now) {
		t.Fatalf("expected Expires %v to be after now %v", parsed, now)
	}

	maxExpiry := now.Add(366 * 24 * time.Hour)
	if parsed.After(maxExpiry) {
		t.Fatalf("expected Expires %v to be no more than 366 days after now %v (max %v)", parsed, now, maxExpiry)
	}
}

func TestSecurityTxt_T005_ExactlyTwoFieldLinesContactAndExpires(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	fields := fieldLines(body)

	if len(fields) != 2 {
		t.Fatalf("expected exactly 2 field lines, got %d: %v", len(fields), fields)
	}

	names := make(map[string]bool)
	for _, l := range fields {
		idx := strings.Index(l, ":")
		if idx < 0 {
			t.Fatalf("expected field line %q to contain a colon", l)
		}
		names[l[:idx]] = true
	}

	if !names["Contact"] || !names["Expires"] {
		t.Fatalf("expected field names to be exactly Contact and Expires, got %v", names)
	}
	if len(names) != 2 {
		t.Fatalf("expected exactly 2 distinct field names, got %v", names)
	}
}

func TestSecurityTxt_T006_FieldLinesMatchNameColonSpaceValueShape(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	fields := fieldLines(body)

	wantNames := map[string]bool{"Contact": false, "Expires": false}

	for _, l := range fields {
		idx := strings.Index(l, ":")
		if idx < 0 {
			t.Fatalf("expected field line %q to contain a colon", l)
		}
		name := l[:idx]
		rest := l[idx+1:]

		if _, ok := wantNames[name]; !ok {
			continue
		}
		wantNames[name] = true

		if !strings.HasPrefix(rest, " ") {
			t.Fatalf("expected field line %q to have a single space after the colon", l)
		}
		value := strings.TrimPrefix(rest, " ")
		if strings.HasPrefix(value, " ") {
			t.Fatalf("expected field line %q to have exactly one space after the colon", l)
		}
		if value == "" {
			t.Fatalf("expected field line %q to have a non-empty value", l)
		}
	}

	if !wantNames["Contact"] || !wantNames["Expires"] {
		t.Fatalf("expected both Contact and Expires field lines to be present and validated, got %v", wantNames)
	}
}

func TestSecurityTxt_T007_UnrelatedPathDoesNotMatch(t *testing.T) {
	r := newSecurityTxtTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected unrelated path /foo to not resolve to the security.txt handler, got status 200")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unmatched route, got %d", rec.Code)
	}
}
