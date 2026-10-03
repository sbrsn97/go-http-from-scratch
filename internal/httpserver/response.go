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

func textResponse(statusCode int, statusText string, body string) Response {
	return Response{
		StatusCode: statusCode,
		StatusText: statusText,
		Headers: map[string]string{
			"Content-Type": "text/plain; charset=utf-8",
		},
		Body: []byte(body),
	}
}

func notFoundResponse() Response {
	return textResponse(
		404,
		"Not Found",
		"404 Not Found\n",
	)
}

func methodNotAllowedResponse() Response {
	return textResponse(
		405,
		"Method Not Allowed",
		"method not allowed\n",
	)
}

func internalServerErrorResponse() Response {
	return textResponse(
		500,
		"Internal Server Error",
		"internal server error\n",
	)
}
