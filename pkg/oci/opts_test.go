package oci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	spec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestPushBlob(t *testing.T) {
	tests := []struct {
		name         string
		opts         PushBlobOptions
		setupServer  func() *httptest.Server
		expectError  bool
		expectedCode int
	}{
		{
			name: "Successful push blob",
			opts: PushBlobOptions{
				Digest: spec.Descriptor{
					Digest: "sha256:1234567890abcdef",
				},
				File:     []byte("test content"),
				Name:     "testblob",
				Insecure: false,
				Tag:      Tag{Host: "localhost", Name: "testblob", Version: "v1"},
			},
			setupServer: func() *httptest.Server {
				// Mock the server responses
				mux := http.NewServeMux()
				mux.HandleFunc("/v2/testblob/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "/upload/location")
					w.WriteHeader(http.StatusAccepted)
				})
				mux.HandleFunc("/upload/location", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				})
				return httptest.NewServer(mux)
			},
			expectError: false,
		},
		{
			name: "Unauthorized push blob",
			opts: PushBlobOptions{
				Digest: spec.Descriptor{
					Digest: "sha256:1234567890abcdef",
				},
				File:     []byte("test content"),
				Name:     "testblob",
				Insecure: false,
				Tag:      Tag{Host: "localhost", Name: "testblob", Version: "v1"},
			},
			setupServer: func() *httptest.Server {
				// Mock the server to return 401 Unauthorized
				mux := http.NewServeMux()
				mux.HandleFunc("/v2/testblob/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				})
				return httptest.NewServer(mux)
			},
			expectError: true,
		},
		{
			name: "Server error on upload",
			opts: PushBlobOptions{
				Digest: spec.Descriptor{
					Digest: "sha256:1234567890abcdef",
				},
				File:     []byte("test content"),
				Name:     "testblob",
				Insecure: false,
				Tag:      Tag{Host: "localhost", Name: "testblob", Version: "v1"},
			},
			setupServer: func() *httptest.Server {
				mux := http.NewServeMux()
				mux.HandleFunc("/v2/testblob/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "/upload/location")
					w.WriteHeader(http.StatusAccepted)
				})
				mux.HandleFunc("/upload/location", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
				return httptest.NewServer(mux)
			},
			expectError: true,
		},
		{
			name: "Invalid URL",
			opts: PushBlobOptions{
				Digest: spec.Descriptor{
					Digest: "sha256:1234567890abcdef",
				},
				File:     []byte("test content"),
				Name:     "testblob",
				Insecure: false,
				Tag:      Tag{Host: ":", Name: "testblob", Version: "v1"}, // Invalid Host
			},
			setupServer: func() *httptest.Server {
				return nil
			},
			expectError: true,
		},
		{
			name: "Insecure connection",
			opts: PushBlobOptions{
				Digest: spec.Descriptor{
					Digest: "sha256:1234567890abcdef",
				},
				File:     []byte("test content"),
				Name:     "testblob",
				Insecure: true,
				Tag:      Tag{Host: "localhost", Name: "testblob", Version: "v1"},
			},
			setupServer: func() *httptest.Server {
				// Similar to successful case but with Insecure flag
				mux := http.NewServeMux()
				mux.HandleFunc("/v2/testblob/blobs/uploads/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "/upload/location")
					w.WriteHeader(http.StatusAccepted)
				})
				mux.HandleFunc("/upload/location", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				})
				return httptest.NewUnstartedServer(mux)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := tt.setupServer()
			if server != nil {
				if tt.opts.Insecure {
					server.Start()
				} else {
					server.StartTLS()
				}
				defer server.Close()
				tt.opts.Tag.Host = server.Listener.Addr().String()
			}

			// Create OciClient
			client := NewOciClient()

			// Call PushBlob
			err := client.PushBlob(tt.opts)
			if (err != nil) != tt.expectError {
				t.Errorf("PushBlob() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
