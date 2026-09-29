package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
)

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

	fmt.Println("method:", method)
	fmt.Println("target:", target)
	fmt.Println("version:", version)

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
