package httpserver

import (
	"bytes"
	"testing"
)

func TestSerializeResponse(t *testing.T) {
	resp := Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: []byte("hello"),
	}

	data := serializeResponse(resp)

	if !bytes.Contains(data, []byte("HTTP/1.1 200 OK\r\n")) {
		t.Error("missing status line")
	}

	if !bytes.Contains(data, []byte("Content-Length: 5\r\n")) {
		t.Error("missing correct Content-Length")
	}

	if !bytes.HasSuffix(data, []byte("\r\n\r\nhello")) {
		t.Error("response body is not serialized correctly")
	}
}

func TestSerializeChunkedResponse(t *testing.T) {
	resp := Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body:    []byte("hello"),
		Chunked: true,
	}

	data := serializeResponse(resp)

	if !bytes.Contains(data, []byte("Transfer-Encoding: chunked\r\n")) {
		t.Error("missing chunked Transfer-Encoding")
	}

	if bytes.Contains(data, []byte("Content-Length:")) {
		t.Error("chunked response must not include Content-Length")
	}

	if !bytes.HasSuffix(data, []byte("5\r\nhello\r\n0\r\n\r\n")) {
		t.Errorf("unexpected chunked body: %q", data)
	}
}
