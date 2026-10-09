package httpserver

import (
	"net"
	"testing"
)

func TestReadRequestFromConnection(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte(
			"POST /echo?x=1 HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"Content-Length: 5\r\n" +
				"\r\n" +
				"hello",
		))
	}()

	req, err := reader.readRequest()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("expected POST, got %q", req.Method)
	}

	if req.Path != "/echo" {
		t.Errorf("expected /echo, got %q", req.Path)
	}

	if req.RawQuery != "x=1" {
		t.Errorf("expected query x=1, got %q", req.RawQuery)
	}

	if string(req.Body) != "hello" {
		t.Errorf("expected body %q, got %q", "hello", string(req.Body))
	}
}

func TestReadRequestAcrossMultipleWrites(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte("POST /echo HTTP/1.1\r\n"))
		_, _ = clientConn.Write([]byte("Host: localhost\r\n"))
		_, _ = clientConn.Write([]byte("Content-Length: 10\r\n"))
		_, _ = clientConn.Write([]byte("\r\n"))
		_, _ = clientConn.Write([]byte("hello"))
		_, _ = clientConn.Write([]byte("world"))
	}()

	req, err := reader.readRequest()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(req.Body) != "helloworld" {
		t.Errorf("expected body %q, got %q", "helloworld", string(req.Body))
	}
}

func TestRequestReaderPreservesNextRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte(
			"GET /hello HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"\r\n" +
				"GET /second HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"\r\n",
		))
	}()

	first, err := reader.readRequest()
	if err != nil {
		t.Fatalf("reading first request: %v", err)
	}

	second, err := reader.readRequest()
	if err != nil {
		t.Fatalf("reading second request: %v", err)
	}

	if first.Path != "/hello" {
		t.Errorf("expected first path /hello, got %q", first.Path)
	}

	if second.Path != "/second" {
		t.Errorf("expected second path /second, got %q", second.Path)
	}
}

func TestReadRequestRejectsOversizedBody(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte(
			"POST /echo HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"Content-Length: 2000000\r\n" +
				"\r\n",
		))
	}()

	_, err := reader.readRequest()

	if err == nil {
		t.Fatal("expected oversized body error")
	}

	if err != errBodyTooLarge {
		t.Fatalf("expected errBodyTooLarge, got %v", err)
	}
}

func TestReadChunkedRequestBody(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte(
			"POST /echo HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"\r\n" +
				"5\r\n" +
				"hello\r\n" +
				"6\r\n" +
				" world\r\n" +
				"0\r\n" +
				"\r\n",
		))
	}()

	req, err := reader.readRequest()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(req.Body) != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", string(req.Body))
	}
}

func TestReadChunkedRequestAcrossMultipleWrites(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		parts := []string{
			"POST /echo HTTP/1.1\r\n",
			"Host: localhost\r\n",
			"Transfer-Encoding: chunked\r\n",
			"\r\n",
			"5\r",
			"\nhel",
			"lo\r\n",
			"6\r\n",
			" world\r",
			"\n0\r\n",
			"\r\n",
		}

		for _, part := range parts {
			_, _ = clientConn.Write([]byte(part))
		}
	}()

	req, err := reader.readRequest()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if string(req.Body) != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", string(req.Body))
	}
}

func TestChunkedRequestPreservesNextRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	reader := newRequestReader(serverConn)

	go func() {
		_, _ = clientConn.Write([]byte(
			"POST /echo HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"\r\n" +
				"5\r\nhello\r\n" +
				"0\r\n\r\n" +
				"GET /hello HTTP/1.1\r\n" +
				"Host: localhost\r\n" +
				"\r\n",
		))
	}()

	first, err := reader.readRequest()
	if err != nil {
		t.Fatalf("reading first request: %v", err)
	}

	second, err := reader.readRequest()
	if err != nil {
		t.Fatalf("reading second request: %v", err)
	}

	if string(first.Body) != "hello" {
		t.Errorf("expected first body %q, got %q", "hello", string(first.Body))
	}

	if second.Path != "/hello" {
		t.Errorf("expected second path /hello, got %q", second.Path)
	}
}
