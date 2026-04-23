package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestParseManifestPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantRepository string
		wantReference  string
		wantOK         bool
	}{
		{
			name:           "tag",
			path:           "/v2/library/ubuntu/manifests/latest",
			wantRepository: "library/ubuntu",
			wantReference:  "latest",
			wantOK:         true,
		},
		{
			name:           "digest",
			path:           "/v2/library/ubuntu/manifests/sha256:abc123",
			wantRepository: "library/ubuntu",
			wantReference:  "sha256:abc123",
			wantOK:         true,
		},
		{
			name:   "missing v2 prefix",
			path:   "/library/ubuntu/manifests/latest",
			wantOK: false,
		},
		{
			name:   "missing manifest segment",
			path:   "/v2/library/ubuntu/tags/list",
			wantOK: false,
		},
		{
			name:   "empty repository",
			path:   "/v2//manifests/latest",
			wantOK: false,
		},
		{
			name:   "empty reference",
			path:   "/v2/library/ubuntu/manifests/",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepository, gotReference, gotOK := parseManifestPath(tt.path)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotRepository != tt.wantRepository || gotReference != tt.wantReference {
				t.Fatalf("parseManifestPath() = (%q, %q), want (%q, %q)", gotRepository, gotReference, tt.wantRepository, tt.wantReference)
			}
		})
	}
}

func TestHandleV2Root(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v2/", nil)
	rec := httptest.NewRecorder()

	handleV2(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Docker-Distribution-API-Version"); got != "registry/2.0" {
		t.Fatalf("Docker-Distribution-API-Version = %q, want registry/2.0", got)
	}
}

func TestHandleV2ManifestMethods(t *testing.T) {
	tests := []struct {
		method     string
		wantStatus int
		wantBody   bool
	}{
		{method: http.MethodGet, wantStatus: http.StatusOK, wantBody: true},
		{method: http.MethodHead, wantStatus: http.StatusOK, wantBody: false},
		{method: http.MethodOptions, wantStatus: http.StatusNoContent, wantBody: false},
		{method: http.MethodPost, wantStatus: http.StatusMethodNotAllowed, wantBody: true},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/v2/library/ubuntu/manifests/latest", nil)
			rec := httptest.NewRecorder()

			handleV2(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			body := rec.Body.String()
			if tt.wantBody && body == "" {
				t.Fatal("body is empty, want response body")
			}
			if !tt.wantBody && body != "" {
				t.Fatalf("body = %q, want empty", body)
			}

			if tt.wantStatus != http.StatusOK {
				return
			}
			if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, indexType) {
				t.Fatalf("Content-Type = %q, want %q", got, indexType)
			}
			if got := resp.Header.Get("Docker-Content-Digest"); got != indexDigest {
				t.Fatalf("Docker-Content-Digest = %q, want %q", got, indexDigest)
			}
		})
	}
}

func TestHandleV2NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v2/library/ubuntu/tags/list", nil)
	rec := httptest.NewRecorder()

	handleV2(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Result().StatusCode, http.StatusNotFound)
	}
}

func TestIndexBodyDigestMatchesHeader(t *testing.T) {
	sum := sha256.Sum256(indexBody())
	got := "sha256:" + hex.EncodeToString(sum[:])

	if got != indexDigest {
		t.Fatalf("computed digest = %q, want %q", got, indexDigest)
	}
}
