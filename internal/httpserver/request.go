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

func readRequest(conn net.Conn) (Request, error) {
	readBuffer := make([]byte, 1024)
	var data []byte
	headerEnd := -1

	for {
		n, err := conn.Read(readBuffer)
		if err != nil {
			return Request{}, err
		}

		data = append(data, readBuffer[:n]...)

		headerEnd = bytes.Index(data, []byte("\r\n\r\n"))
		if headerEnd != -1 {
			break
		}
	}

	if headerEnd == -1 {
		return Request{}, errors.New("missing header terminator")
	}

	headerBlock := data[:headerEnd]
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

	bodyStart := headerEnd + 4 // Skip the "\r\n\r\n" sequence
	bodyBytesAvailable := len(data) - bodyStart

	for bodyBytesAvailable < length {
		n, err := conn.Read(readBuffer)
		if err != nil {
			return Request{}, err
		}

		data = append(data, readBuffer[:n]...)
		bodyBytesAvailable = len(data) - bodyStart
	}

	body := data[bodyStart : bodyStart+length]

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
