// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

// securityTxtLine is one parsed "Name: value" line of a security.txt-shaped body.
type securityTxtLine struct {
	name  string
	value string
}

// parseSecurityTxtLines splits body into non-blank lines and parses each as
// "name: value", returning an error for any line that doesn't match that shape.
// Shared by tests that need to inspect field names/values (T-006, T-009) to
// avoid duplicating the line-parsing logic.
func parseSecurityTxtLines(t *testing.T, body string) []securityTxtLine {
	t.Helper()

	var lines []securityTxtLine
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			t.Fatalf("body line %q does not match the %q shape", line, "name: value")
			continue
		}
		name := line[:idx]
		value := strings.TrimSpace(line[idx+1:])
		lines = append(lines, securityTxtLine{name: name, value: value})
	}
	return lines
}

// newSecurityTxtRouter builds a router with the security.txt route registered
// under base, mirroring how main() wires registerSecurityTxt(r, baseUrl).
func newSecurityTxtRouter(base string) *mux.Router {
	r := mux.NewRouter()
	registerSecurityTxt(r, base)
	return r
}

// doRequest dispatches method/path through r and returns the recorded response.
func doRequest(r *mux.Router, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// T-001: GET /.well-known/security.txt returns 200.
func TestSecurityTxt_T001_GetReturns200(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// T-002: Content-Type is exactly "text/plain; charset=utf-8".
func TestSecurityTxt_T002_ContentTypeIsExact(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	const want = "text/plain; charset=utf-8"
	if got := rec.Header().Get("Content-Type"); got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
}

// T-003: body contains the exact Contact line.
func TestSecurityTxt_T003_ContactLinePresent(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	const want = "Contact: mailto:security@example.com"
	body := rec.Body.String()
	found := false
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimRight(line, "\r") == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("body %q does not contain the exact line %q", body, want)
	}
}

// T-004: exactly one Expires: line, valid RFC 3339, strictly in the future.
func TestSecurityTxt_T004_ExpiresPresentValidFuture(t *testing.T) {
	r := newSecurityTxtRouter("")

	now := time.Now()
	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	var expiresValues []string
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "Expires:") {
			expiresValues = append(expiresValues, strings.TrimSpace(strings.TrimPrefix(line, "Expires:")))
		}
	}

	if len(expiresValues) != 1 {
		t.Fatalf("found %d Expires: lines, want exactly 1 (values: %v)", len(expiresValues), expiresValues)
	}

	parsed, err := time.Parse(time.RFC3339, expiresValues[0])
	if err != nil {
		t.Fatalf("Expires value %q did not parse as RFC3339: %v", expiresValues[0], err)
	}
	if !parsed.After(now) {
		t.Fatalf("Expires value %v is not strictly after now (%v)", parsed, now)
	}
}

// T-006: every non-blank body line's field name is Contact or Expires only.
func TestSecurityTxt_T006_OnlyContactAndExpiresFields(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	lines := parseSecurityTxtLines(t, rec.Body.String())
	for _, l := range lines {
		if l.name != "Contact" && l.name != "Expires" {
			t.Fatalf("unexpected field %q in body (only Contact/Expires allowed)", l.name)
		}
	}
}

// T-005: Expires is within +-5 days of exactly one year from now.
func TestSecurityTxt_T005_ExpiresIsApproximatelyOneYearOut(t *testing.T) {
	r := newSecurityTxtRouter("")

	now := time.Now()
	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	var expiresValue string
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "Expires:") {
			expiresValue = strings.TrimSpace(strings.TrimPrefix(line, "Expires:"))
			break
		}
	}
	if expiresValue == "" {
		t.Fatalf("no Expires: line found in body %q", rec.Body.String())
	}

	parsed, err := time.Parse(time.RFC3339, expiresValue)
	if err != nil {
		t.Fatalf("Expires value %q did not parse as RFC3339: %v", expiresValue, err)
	}

	target := now.AddDate(1, 0, 0)
	windowStart := target.Add(-5 * 24 * time.Hour)
	windowEnd := target.Add(5 * 24 * time.Hour)
	if parsed.Before(windowStart) || parsed.After(windowEnd) {
		t.Fatalf("Expires %v is outside the window [%v, %v]", parsed, windowStart, windowEnd)
	}
}

// T-007: POST to the path does not return 200 (405 expected, GET/HEAD-only route).
func TestSecurityTxt_T007_PostIsNotServedAs200(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodPost, "/.well-known/security.txt")

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want != %d (expected %d)", rec.Code, http.StatusOK, http.StatusMethodNotAllowed)
	}
	if rec.Code != http.StatusMethodNotAllowed {
		t.Logf("status = %d, expected %d for a GET/HEAD-only mux route", rec.Code, http.StatusMethodNotAllowed)
	}
}

// T-008: the route respects a base prefix, passed directly to
// registerSecurityTxt rather than via the package-level baseUrl var.
func TestSecurityTxt_T008_RouteRespectsBaseUrl(t *testing.T) {
	const base = "/some-base"
	r := newSecurityTxtRouter(base)

	prefixed := doRequest(r, http.MethodGet, base+"/.well-known/security.txt")
	if prefixed.Code != http.StatusOK {
		t.Fatalf("prefixed path status = %d, want %d", prefixed.Code, http.StatusOK)
	}

	unprefixed := doRequest(r, http.MethodGet, "/.well-known/security.txt")
	if unprefixed.Code == http.StatusOK {
		t.Fatalf("unprefixed path status = %d, want != %d when baseUrl is set", unprefixed.Code, http.StatusOK)
	}
}

// T-009: body is valid UTF-8 and every non-blank line matches "name: value".
func TestSecurityTxt_T009_BodyIsValidUtf8NameValueLines(t *testing.T) {
	r := newSecurityTxtRouter("")

	rec := doRequest(r, http.MethodGet, "/.well-known/security.txt")

	body := rec.Body.String()
	if !utf8.ValidString(body) {
		t.Fatalf("body is not valid UTF-8: %q", body)
	}

	// parseSecurityTxtLines fails the test if any non-blank line doesn't match
	// the "name: value" shape.
	parseSecurityTxtLines(t, body)
}
