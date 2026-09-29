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

	readBuffer := make([]byte, 1024)
	var data []byte

	for {
		n, err := conn.Read(readBuffer)
		if err != nil {
			fmt.Println("Read error:", err)
			return
		}

		data = append(data, readBuffer[:n]...)

		fmt.Printf("read %d bytes, total %d bytes\n", n, len(data))

		if bytes.Contains(data, []byte("\r\n\r\n")) {
			break
		}
	}

	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd == -1 {
		fmt.Println("Invalid request: missing request line terminator")
		return
	}

	requestLine := data[:lineEnd]

	method, target, version, err := parseRequestLine(requestLine)
	if err != nil {
		fmt.Println("Invalid request:", err)
		return
	}

	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		fmt.Println("Invalid request: missing header terminator")
		return
	}

	headerBlock := data[:headerEnd]

	headerLines := bytes.Split(headerBlock, []byte("\r\n"))

	if len(headerLines) < 1 {
		fmt.Println("Invalid request: empty request")
		return
	}

	headers, err := parseHeaders(headerLines[1:])
	if err != nil {
		fmt.Println("Invalid request:", err)
		return
	}

	length, err := contentLength(headers)
	if err != nil {
		fmt.Println("Invalid request:", err)
		return
	}

	bodyStart := headerEnd + 4 // Skip the "\r\n\r\n" sequence
	bodyBytesAvailable := len(data) - bodyStart

	for bodyBytesAvailable < length {
		n, err := conn.Read(readBuffer)
		if err != nil {
			fmt.Println("read body error:", err)
			return
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

	fmt.Println("method: ", req.Method)
	fmt.Println("target: ", req.Target)
	fmt.Println("version: ", req.Version)
	fmt.Println("headers: ", req.Headers)
	fmt.Println("body: ", string(req.Body))

	host, ok := headers["host"]
	if !ok {
		fmt.Println("Invalid request: missing Host header")
		return
	}

	fmt.Println("Host: ", host)

	_, err = conn.Write([]byte("received\n"))
	if err != nil {
		fmt.Println("Write error:", err)
	}
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
