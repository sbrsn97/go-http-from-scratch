package httpserver

import (
	"bytes"
	"testing"
)

func TestSerializeResponse(t *testing.T) {
	resp := Response{
		StatusCode: 200,
		StatusText: "OK",
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
