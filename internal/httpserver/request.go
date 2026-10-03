package httpserver

import (
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
)

type Request struct {
	Method   string
	Target   string
	Path     string
	RawQuery string
	Version  string
	Headers  map[string]string
	Body     []byte
}

type requestReader struct {
	conn net.Conn
	data []byte
}

const (
	maxHeaderBytes = 16 * 1024       // 16 KB
	maxBodyBytes   = 1 * 1024 * 1024 // 1 MB
)

var (
	errHeaderTooLarge = errors.New("request headers too large")
	errBodyTooLarge   = errors.New("request body too large")
)

func requestHeaderFieldsTooLargeResponse() Response {
	return textResponse(
		431,
		"Request Header Fields Too Large",
		"request headers too large\n",
	)
}

func contentTooLargeResponse() Response {
	return textResponse(
		413,
		"Content Too Large",
		"request body too large\n",
	)
}

func newRequestReader(conn net.Conn) *requestReader {
	return &requestReader{
		conn: conn,
	}
}

func (r *requestReader) readRequest() (Request, error) {
	readBuffer := make([]byte, 1024)

	headerEnd := bytes.Index(r.data, []byte("\r\n\r\n"))

	for headerEnd == -1 {
		if len(r.data) > maxHeaderBytes {
			return Request{}, errHeaderTooLarge
		}

		n, err := r.conn.Read(readBuffer)
		if err != nil {
			return Request{}, err
		}

		r.data = append(r.data, readBuffer[:n]...)
		headerEnd = bytes.Index(r.data, []byte("\r\n\r\n"))
	}

	if headerEnd+4 > maxHeaderBytes {
		return Request{}, errHeaderTooLarge
	}

	headerBlock := r.data[:headerEnd]
	headerLines := bytes.Split(headerBlock, []byte("\r\n"))

	if len(headerLines) < 1 {
		return Request{}, errors.New("empty request")
	}

	method, target, version, err := parseRequestLine(headerLines[0])
	if err != nil {
		return Request{}, err
	}

	path, rawQuery := parseRequestTarget(target)

	headers, err := parseHeaders(headerLines[1:])
	if err != nil {
		return Request{}, err
	}

	if _, ok := headers["host"]; !ok {
		return Request{}, errors.New("missing host header")
	}

	length, err := contentLength(headers)
	if err != nil {
		return Request{}, err
	}

	if length > maxBodyBytes {
		return Request{}, errBodyTooLarge
	}

	bodyStart := headerEnd + 4
	requestEnd := bodyStart + length

	for len(r.data) < requestEnd {
		n, err := r.conn.Read(readBuffer)
		if err != nil {
			return Request{}, err
		}

		r.data = append(r.data, readBuffer[:n]...)
	}

	body := append([]byte(nil), r.data[bodyStart:requestEnd]...)

	req := Request{
		Method:   method,
		Target:   target,
		Path:     path,
		RawQuery: rawQuery,
		Version:  version,
		Headers:  headers,
		Body:     body,
	}

	r.data = r.data[requestEnd:]

	return req, nil
}

func parseRequestLine(line []byte) (string, string, string, error) {
	parts := bytes.Split(line, []byte(" "))

	if len(parts) != 3 {
		return "", "", "", errors.New("request line must contain exactly three parts")
	}

	method := string(parts[0])

	target := string(parts[1])
	version := string(parts[2])
	if version != "HTTP/1.1" {
		return "", "", "", errors.New("unsupported HTTP version: " + version)
	}

	return method, target, version, nil
}

func parseRequestTarget(target string) (string, string) {
	index := strings.Index(target, "?")

	if index == -1 {
		return target, ""
	}

	path := target[:index]
	rawQuery := target[index+1:]

	return path, rawQuery
}

func parseHeaders(lines [][]byte) (map[string]string, error) {
	headers := make(map[string]string)

	for _, line := range lines {
		parts := bytes.SplitN(line, []byte(":"), 2)

		if len(parts) != 2 {
			return nil, errors.New("invalid header line: " + string(line))
		}

		name := strings.ToLower(string(bytes.TrimSpace(parts[0])))
		value := string(bytes.TrimSpace(parts[1]))

		if name == "" {
			return nil, errors.New("header name cannot be empty")
		}

		headers[name] = value
	}

	return headers, nil
}

func contentLength(headers map[string]string) (int, error) {
	value, ok := headers["content-length"]
	if !ok {
		return 0, nil
	}

	length, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New("invalid Content-Length")
	}

	if length < 0 {
		return 0, errors.New("Content-Length cannot be negative")
	}

	return length, nil
}
