package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter_DocumentCacheControl(t *testing.T) {
	router := NewRouter(nil)

	testCases := []struct {
		name string
		path string
	}{
		{name: "Root Document", path: "/"},
		{name: "Explicit Index Document", path: "/index.html"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", tc.path, rec.Code)
			}

			cacheControl := rec.Header().Get("Cache-Control")
			expectedCacheControl := "no-cache, no-store, must-revalidate"
			if cacheControl != expectedCacheControl {
				t.Errorf("expected Cache-Control '%s', got '%s'", expectedCacheControl, cacheControl)
			}

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Errorf("expected Content-Type to contain text/html, got '%s'", contentType)
			}

			// Verify essential security headers
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("expected X-Content-Type-Options 'nosniff', got '%s'", rec.Header().Get("X-Content-Type-Options"))
			}
			if rec.Header().Get("X-Frame-Options") != "DENY" {
				t.Errorf("expected X-Frame-Options 'DENY', got '%s'", rec.Header().Get("X-Frame-Options"))
			}
			if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
				t.Errorf("expected Referrer-Policy 'strict-origin-when-cross-origin', got '%s'", rec.Header().Get("Referrer-Policy"))
			}

			// Ensure response contains landing page elements
			body := rec.Body.String()
			if !strings.Contains(body, "GoodAI Monitor") {
				t.Errorf("expected body to contain 'GoodAI Monitor'")
			}
			if !strings.Contains(body, "style.css?v=") {
				t.Errorf("expected HTML to contain versioned style.css reference")
			}
			if !strings.Contains(body, "app.js?v=") {
				t.Errorf("expected HTML to contain versioned app.js reference")
			}
		})
	}
}

func TestRouter_StaticAssetsCacheControl(t *testing.T) {
	router := NewRouter(nil)

	testCases := []struct {
		name                string
		path                string
		expectedContentType string
	}{
		{
			name:                "Unversioned CSS Asset",
			path:                "/style.css",
			expectedContentType: "text/css",
		},
		{
			name:                "Versioned CSS Asset",
			path:                "/style.css?v=20261014-01",
			expectedContentType: "text/css",
		},
		{
			name:                "Unversioned JS Asset",
			path:                "/app.js",
			expectedContentType: "javascript",
		},
		{
			name:                "Versioned JS Asset",
			path:                "/app.js?v=20261014-01",
			expectedContentType: "javascript",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", tc.path, rec.Code)
			}

			cacheControl := rec.Header().Get("Cache-Control")
			expectedCacheControl := "public, max-age=86400, must-revalidate"
			if cacheControl != expectedCacheControl {
				t.Errorf("expected Cache-Control '%s', got '%s'", expectedCacheControl, cacheControl)
			}

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, tc.expectedContentType) {
				t.Errorf("expected Content-Type to contain '%s', got '%s'", tc.expectedContentType, contentType)
			}

			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("expected X-Content-Type-Options 'nosniff', got '%s'", rec.Header().Get("X-Content-Type-Options"))
			}

			if rec.Body.Len() == 0 {
				t.Errorf("expected non-empty response body for %s", tc.path)
			}
		})
	}
}

func TestRouter_SecurityHeadersAcrossEndpoints(t *testing.T) {
	router := NewRouter(nil)

	paths := []string{"/", "/style.css", "/app.js", "/healthz"}

	for _, p := range paths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("path %s missing X-Content-Type-Options: nosniff", p)
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("path %s missing X-Frame-Options: DENY", p)
		}
		if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
			t.Errorf("path %s missing Referrer-Policy: strict-origin-when-cross-origin", p)
		}
	}
}

func TestRouter_SecurityEndpoints(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	router := NewRouter(db)

	// Unauthenticated requests should return 401 Unauthorized
	secEndpoints := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/security/reports"},
		{method: http.MethodPost, path: "/api/v1/security/audit"},
		{method: http.MethodGet, path: "/api/v1/security/reports/1"},
		{method: http.MethodGet, path: "/api/v1/security/reports/1/export"},
	}

	for _, ep := range secEndpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for %s %s without auth, got %d", ep.method, ep.path, rec.Code)
		}
	}
}

