package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/avalanchian/learn-backend/middleware/clientmw"
)

var client *http.Client
var server *httptest.Server

func TestMain(m *testing.M) {
	router := BuildRouter()

	server = httptest.NewServer(router)
	client = server.Client()

	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}

	client.Transport = clientmw.Log(clientmw.Trace(client.Transport))

	code := m.Run()
	server.Close()
	os.Exit(code)
}

func TestNotFound(t *testing.T) {
	for _, tt := range []struct {
		method, path string
		wantStatus   int
	}{
		{"DELETE", "/", http.StatusMethodNotAllowed},
		{"GET", "/notfound", http.StatusNotFound},
		{"GET", "/chess/replay/efronlicht/bobross/1234", http.StatusNotFound},
	} {
		req, _ := http.NewRequest(tt.method, server.URL+tt.path, nil)

		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("client.Do(%q, %q) returned error: %v", tt.method, tt.path, err)
		} else if resp.StatusCode != tt.wantStatus {
			t.Errorf("client.Do(%q, %q) returned status %d, want %d", tt.method, tt.path, resp.StatusCode, tt.wantStatus)
		}
	}
}

func TestGraduation(t *testing.T) {
	defer server.Close()

	for _, tt := range []struct {
		method, path     string
		body             map[string]any
		queries          map[string]string
		wantStatus       int
		wantBodyContains []string
	}{
		{
			method:           "GET",
			path:             "/",
			wantStatus:       http.StatusOK,
			wantBodyContains: []string{"Hello, World!"},
		},
		{
			method:           "GET",
			path:             "/panic",
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: []string{"internal server error"},
		},
		{
			method:           "POST",
			path:             "/greet/json",
			body:             map[string]any{"first": "Raphael", "last": "Frasca", "age": 10},
			wantStatus:       http.StatusForbidden,
			wantBodyContains: []string{"forbidden"},
		},
		{
			method:           "POST",
			path:             "/greet/json",
			body:             map[string]any{"first": "Matthew", "last": "McRobie", "age": 37},
			wantStatus:       http.StatusOK,
			wantBodyContains: []string{"Matthew", "McRobie", "adult"},
		},
		{
			method:     "GET",
			path:       "/time",
			queries:    map[string]string{"tz": "America/New_York"},
			wantStatus: http.StatusOK,
		},
		{
			method:     "GET",
			path:       "/time",
			queries:    map[string]string{"tz": "invalid"},
			wantStatus: http.StatusBadRequest,
		},
	} {
		tt := tt
		if len(tt.queries) != 0 {
			query := make(url.Values, len(tt.queries))
			for k, v := range tt.queries {
				query.Set(k, v)
			}
			tt.path += "?" + query.Encode()
		}

		name := fmt.Sprintf("%s%s -> %d-%s", tt.method, tt.path, tt.wantStatus, strings.ReplaceAll(http.StatusText(tt.wantStatus), " ", "-"))

		t.Run(name, func(t *testing.T) {
			path := server.URL + tt.path

			var body io.Reader
			if tt.body != nil {
				b, _ := json.Marshal(tt.body)
				body = bytes.NewReader(b)
			}

			req, err := http.NewRequestWithContext(context.TODO(), tt.method, path, body)
			if err != nil {
				t.Errorf("http.NewRequestWithContext(%q, %q, %v) returned error: %v", tt.method, path, body, err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("client.Do(%q, %q) returned error: %v", tt.method, tt.path, err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("router.ServeHTTP(%q, %q) returned status %d, want %d", tt.method, tt.path, resp.StatusCode, tt.wantStatus)
			}

			bodyBytes, _ := io.ReadAll(resp.Body)

			resp.Body.Close()

			for _, want := range tt.wantBodyContains {
				if !strings.Contains(string(bodyBytes), want) {
					t.Errorf("router.ServeHTTP(%q, %q) returned body %s, want body to contain %s", tt.method, tt.path, bodyBytes, want)
				}
			}
		})
	}
}
