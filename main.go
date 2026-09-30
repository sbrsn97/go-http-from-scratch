package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string]string
	Body    []byte
}

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       []byte
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening on: 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	req, err := readRequest(conn)
	if err != nil {
		fmt.Println("invalid request:", err)
		return
	}

	fmt.Println("method:", req.Method)
	fmt.Println("target:", req.Target)
	fmt.Println("version:", req.Version)
	fmt.Println("headers:", req.Headers)
	fmt.Println("body:", string(req.Body))

	resp := handleRequest(req)

	err = writeResponse(conn, resp)
	if err != nil {
		fmt.Println("write response error:", err)
	}
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
		Method:  method,
		Target:  target,
		Version: version,
		Headers: headers,
		Body:    body,
	}

	return req, nil
}

func handleRequest(req Request) Response {
	if req.Target == "/hello" {
		return Response{
			StatusCode: 200,
			StatusText: "OK",
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       []byte("hello from our HTTP server\n"),
		}
	}

	return Response{
		StatusCode: 404,
		StatusText: "Not Found",
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte("404 Not Found"),
	}
}

func writeResponse(conn net.Conn, resp Response) error {
	responseBytes := serializeResponse(resp)

	_, err := conn.Write(responseBytes)
	return err
}

func parseRequestLine(line []byte) (string, string, string, error) {
	parts := bytes.Split(line, []byte(" "))

	if len(parts) != 3 {
		return "", "", "", errors.New("request line must contain exactly three parts")
	}

	method := string(parts[0])
	if method != "GET" && method != "POST" {
		return "", "", "", errors.New("unsupported HTTP method: " + method)
	}

	target := string(parts[1])
	version := string(parts[2])
	if version != "HTTP/1.1" {
		return "", "", "", errors.New("unsupported HTTP version: " + version)
	}

	return method, target, version, nil
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

func serializeResponse(resp Response) []byte {
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}

	var data []byte

	statusLine := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n",
		resp.StatusCode,
		resp.StatusText,
	)

	data = append(data, []byte(statusLine)...)

	resp.Headers["Content-Length"] = strconv.Itoa(len(resp.Body))

	for name, value := range resp.Headers {
		headerLine := fmt.Sprintf("%s: %s\r\n", name, value)
		data = append(data, []byte(headerLine)...)
	}

	data = append(data, []byte("\r\n")...)
	data = append(data, resp.Body...)

	return data
}
