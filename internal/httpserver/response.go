package httpserver

import (
	"fmt"
	"net"
	"strconv"
)

type Response struct {
	StatusCode int
	StatusText string
	Headers    map[string]string
	Body       []byte
}

func handleRequest(req Request) Response {
	if req.Path == "/hello" {
		if req.Method != "GET" {
			return Response{
				StatusCode: 405,
				StatusText: "Method Not Allowed",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte("method not allowed\n"),
			}
		}

		return Response{
			StatusCode: 200,
			StatusText: "OK",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: []byte("hello from our HTTP server\n"),
		}
	}

	if req.Path == "/echo" {
		if req.Method != "POST" {
			return Response{
				StatusCode: 405,
				StatusText: "Method Not Allowed",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte("method not allowed\n"),
			}
		}

		return Response{
			StatusCode: 200,
			StatusText: "OK",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: req.Body,
		}
	}

	return Response{
		StatusCode: 404,
		StatusText: "Not Found",
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: []byte("404 Not Found\n"),
	}
}

func writeResponse(conn net.Conn, resp Response) error {
	responseBytes := serializeResponse(resp)

	_, err := conn.Write(responseBytes)
	return err
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
