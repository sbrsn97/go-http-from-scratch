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
	errHeaderTooLarge       = errors.New("request headers too large")
	errBodyTooLarge         = errors.New("request body too large")
	errInvalidChunkedBody   = errors.New("invalid chunked body")
	errAmbiguousBodyFraming = errors.New("ambiguous request body framing")
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
	headerEnd := bytes.Index(r.data, []byte("\r\n\r\n"))

	for headerEnd == -1 {
		if len(r.data) > maxHeaderBytes {
			return Request{}, errHeaderTooLarge
		}

		if err := r.readMore(); err != nil {
			return Request{}, err
		}

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

	path, rawQuery, err := parseRequestTarget(target)
	if err != nil {
		return Request{}, err
	}

	headers, err := parseHeaders(headerLines[1:])
	if err != nil {
		return Request{}, err
	}

	if _, ok := headers["host"]; !ok {
		return Request{}, errors.New("missing host header")
	}

	_, hasContentLength := headers["content-length"]
	transferEncoding, hasTransferEncoding := headers["transfer-encoding"]

	if hasContentLength && hasTransferEncoding {
		return Request{}, errAmbiguousBodyFraming
	}

	bodyStart := headerEnd + 4

	var body []byte
	var requestEnd int

	switch {
	case hasTransferEncoding:
		if !strings.EqualFold(strings.TrimSpace(transferEncoding), "chunked") {
			return Request{}, errors.New("unsupported Transfer-Encoding")
		}

		chunkedBody, end, err := r.readChunkedBody(bodyStart)
		if err != nil {
			return Request{}, err
		}

		body = chunkedBody
		requestEnd = end

	case hasContentLength:
		length, err := contentLength(headers)
		if err != nil {
			return Request{}, err
		}

		if length > maxBodyBytes {
			return Request{}, errBodyTooLarge
		}

		requestEnd = bodyStart + length

		for len(r.data) < requestEnd {
			if err := r.readMore(); err != nil {
				return Request{}, err
			}
		}

		body = append([]byte(nil), r.data[bodyStart:requestEnd]...)

	default:
		body = []byte{}
		requestEnd = bodyStart
	}

	r.data = r.data[requestEnd:]

	req := Request{
		Method:   method,
		Target:   target,
		Path:     path,
		RawQuery: rawQuery,
		Version:  version,
		Headers:  headers,
		Body:     body,
	}

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

func parseRequestTarget(target string) (string, string, error) {
	if target == "" {
		return "", "", errors.New("empty request target")
	}

	if !strings.HasPrefix(target, "/") {
		return "", "", errors.New("unsupported request target")
	}

	index := strings.Index(target, "?")
	if index == -1 {
		return target, "", nil
	}

	path := target[:index]
	rawQuery := target[index+1:]

	return path, rawQuery, nil
}

func parseHeaders(lines [][]byte) (map[string]string, error) {
	headers := make(map[string]string)

	for _, line := range lines {
		if len(line) == 0 {
			return nil, errors.New("unexpected empty header line")
		}

		parts := bytes.SplitN(line, []byte(":"), 2)

		if len(parts) != 2 {
			return nil, errors.New("invalid header line")
		}

		rawName := parts[0]

		if len(rawName) == 0 {
			return nil, errors.New("header name cannot be empty")
		}

		if bytes.ContainsAny(rawName, " \t") {
			return nil, errors.New("invalid whitespace in header name")
		}

		name := strings.ToLower(string(rawName))
		value := string(bytes.TrimSpace(parts[1]))

		if name == "" {
			return nil, errors.New("header name cannot be empty")
		}

		if name == "content-length" {
			if _, exists := headers[name]; exists {
				return nil, errors.New("duplicate Content-Length")
			}
		}

		headers[name] = value
	}

	return headers, nil
}

func (r *requestReader) readChunkedBody(bodyStart int) ([]byte, int, error) {
	position := bodyStart
	var body []byte

	for {
		lineEnd := bytes.Index(r.data[position:], []byte("\r\n"))

		for lineEnd == -1 {
			if err := r.readMore(); err != nil {
				return nil, 0, err
			}

			lineEnd = bytes.Index(r.data[position:], []byte("\r\n"))
		}

		lineEnd += position
		sizeLine := r.data[position:lineEnd]

		semicolon := bytes.IndexByte(sizeLine, ';')
		if semicolon != -1 {
			sizeLine = sizeLine[:semicolon]
		}

		if len(sizeLine) == 0 {
			return nil, 0, errInvalidChunkedBody
		}

		chunkSize, err := strconv.ParseUint(string(sizeLine), 16, 64)
		if err != nil {
			return nil, 0, errInvalidChunkedBody
		}

		position = lineEnd + 2

		if chunkSize == 0 {
			break
		}

		if chunkSize > uint64(maxBodyBytes-len(body)) {
			return nil, 0, errBodyTooLarge
		}

		chunkEnd := position + int(chunkSize)
		requiredEnd := chunkEnd + 2

		for len(r.data) < requiredEnd {
			if err := r.readMore(); err != nil {
				return nil, 0, err
			}
		}

		if !bytes.Equal(r.data[chunkEnd:requiredEnd], []byte("\r\n")) {
			return nil, 0, errInvalidChunkedBody
		}

		body = append(body, r.data[position:chunkEnd]...)

		position = requiredEnd
	}

	for {
		lineEnd := bytes.Index(r.data[position:], []byte("\r\n"))

		for lineEnd == -1 {
			if err := r.readMore(); err != nil {
				return nil, 0, err
			}

			lineEnd = bytes.Index(r.data[position:], []byte("\r\n"))
		}

		lineEnd += position

		if lineEnd == position {
			position += 2
			break
		}

		position = lineEnd + 2
	}

	return body, position, nil
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

func (r *requestReader) readMore() error {
	buffer := make([]byte, 1024)

	n, err := r.conn.Read(buffer)
	if err != nil {
		return err
	}

	r.data = append(r.data, buffer[:n]...)
	return nil
}
