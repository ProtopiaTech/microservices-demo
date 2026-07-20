// Copyright 2024 Google LLC
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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

const securityTxtPath = "/.well-known/security.txt"

// newSecurityTxtRouter builds a minimal router with only the security.txt
// route registered, driving requests straight to the real securityTxtHandler
// from main.go.
func newSecurityTxtRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc(securityTxtPath, securityTxtHandler)
	return r
}

// newSecurityTxtRouterWithBaseURL builds a router that also registers other
// routes under a non-empty baseUrl prefix, to pin that security.txt is
// reachable at the literal root regardless of that prefix (T-006).
func newSecurityTxtRouterWithBaseURL(baseUrl string) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc(baseUrl+"/shop/", func(w http.ResponseWriter, _ *http.Request) {})
	r.HandleFunc(securityTxtPath, securityTxtHandler)
	return r
}

func TestT001_HappyPath_200OK(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestT002_ContentTypeIsExactlyTextPlainCharsetUTF8(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	got := rec.Header().Get("Content-Type")
	want := "text/plain; charset=utf-8"
	if got != want {
		t.Fatalf("expected Content-Type %q, got %q", want, got)
	}
}

func TestT003_ContactFieldPresentWithPlaceholderValue(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	want := "Contact: mailto:security@example.com"
	body := rec.Body.String()
	found := false
	for _, line := range strings.Split(body, "\n") {
		if line == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a line exactly equal to %q in body:\n%s", want, body)
	}
}

func TestT004_ExpiresFieldPresentOnceValidAndInFuture(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	var expiresLines []string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "Expires:") {
			expiresLines = append(expiresLines, line)
		}
	}
	if len(expiresLines) != 1 {
		t.Fatalf("expected exactly one Expires: line, got %d: %v", len(expiresLines), expiresLines)
	}

	value := strings.TrimPrefix(expiresLines[0], "Expires: ")
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected Expires value to parse as RFC3339, got %q: %v", value, err)
	}

	now := time.Now()
	if !parsed.After(now) {
		t.Fatalf("expected Expires %v to be after now %v", parsed, now)
	}
	if !parsed.Before(now.AddDate(1, 0, 0)) {
		t.Fatalf("expected Expires %v to be before one year from now %v", parsed, now.AddDate(1, 0, 0))
	}
}

func TestT005_OnlyRequiredFieldsAppear(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	body := rec.Body.String()
	fieldLine := regexp.MustCompile(`^(Contact|Expires): `)
	disallowedPrefixes := []string{
		"Policy:", "Preferred-Languages:", "Encryption:", "Acknowledgments:", "Canonical:", "Hiring:",
	}

	for _, line := range strings.Split(body, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !fieldLine.MatchString(line) {
			t.Fatalf("expected line to match ^(Contact|Expires): , got %q", line)
		}
		for _, prefix := range disallowedPrefixes {
			if strings.HasPrefix(line, prefix) {
				t.Fatalf("expected no line to begin with %q, but found %q", prefix, line)
			}
		}
	}
}

func TestT006_RouteResolvesAtLiteralRootIndependentOfBaseURL(t *testing.T) {
	router := newSecurityTxtRouterWithBaseURL("/shop")
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for literal path regardless of baseUrl, got %d", rec.Code)
	}
}

func TestT007_GETIsAccepted(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodGet, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET, got %d", rec.Code)
	}
}

func TestT008_POSTIsUnrestricted(t *testing.T) {
	router := newSecurityTxtRouter()
	req := httptest.NewRequest(http.MethodPost, securityTxtPath, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for POST (method unrestricted), got %d", rec.Code)
	}
}
