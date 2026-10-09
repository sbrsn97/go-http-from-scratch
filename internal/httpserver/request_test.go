package httpserver

import "testing"

func TestParseRequestLine(t *testing.T) {
	method, target, version, err := parseRequestLine(
		[]byte("GET /hello HTTP/1.1"),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if method != "GET" {
		t.Errorf("expected method GET, got %q", method)
	}

	if target != "/hello" {
		t.Errorf("expected target /hello, got %q", target)
	}

	if version != "HTTP/1.1" {
		t.Errorf("expected version HTTP/1.1, got %q", version)
	}
}

func TestParseRequestTarget(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		wantPath  string
		wantQuery string
		wantErr   bool
	}{
		{
			name:      "path only",
			target:    "/hello",
			wantPath:  "/hello",
			wantQuery: "",
		},
		{
			name:      "path with query",
			target:    "/hello?name=sabri",
			wantPath:  "/hello",
			wantQuery: "name=sabri",
		},
		{
			name:    "unsupported target",
			target:  "hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, query, err := parseRequestTarget(tt.target)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if path != tt.wantPath {
				t.Errorf("expected path %q, got %q", tt.wantPath, path)
			}

			if query != tt.wantQuery {
				t.Errorf("expected query %q, got %q", tt.wantQuery, query)
			}
		})
	}
}

func TestParseHeadersRejectsDuplicateContentLength(t *testing.T) {
	lines := [][]byte{
		[]byte("Host: localhost"),
		[]byte("Content-Length: 5"),
		[]byte("Content-Length: 10"),
	}

	_, err := parseHeaders(lines)

	if err == nil {
		t.Fatal("expected duplicate Content-Length to be rejected")
	}
}

func TestParseHeadersNormalizesNames(t *testing.T) {
	lines := [][]byte{
		[]byte("Host: localhost"),
		[]byte("User-Agent: test-client"),
	}

	headers, err := parseHeaders(lines)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if headers["host"] != "localhost" {
		t.Errorf("expected host header to be normalized")
	}

	if headers["user-agent"] != "test-client" {
		t.Errorf("expected user-agent header to be normalized")
	}
}

func TestContentLength(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    int
		wantErr bool
	}{
		{
			name:    "missing",
			headers: map[string]string{},
			want:    0,
		},
		{
			name: "valid",
			headers: map[string]string{
				"content-length": "5",
			},
			want: 5,
		},
		{
			name: "invalid",
			headers: map[string]string{
				"content-length": "banana",
			},
			wantErr: true,
		},
		{
			name: "negative",
			headers: map[string]string{
				"content-length": "-1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := contentLength(tt.headers)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

func TestParseHeadersRejectsWhitespaceBeforeColon(t *testing.T) {
	lines := [][]byte{
		[]byte("Host : localhost"),
	}

	_, err := parseHeaders(lines)
	if err == nil {
		t.Fatal("expected invalid header name to be rejected")
	}
}

func TestParseRequestTargetRejectsNonOriginForm(t *testing.T) {
	_, _, err := parseRequestTarget("http://example.com/test")

	if err == nil {
		t.Fatal("expected absolute-form target to be rejected")
	}
}
