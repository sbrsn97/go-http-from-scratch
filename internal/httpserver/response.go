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
	Chunked    bool
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

	if resp.Chunked {
		resp.Headers["Transfer-Encoding"] = "chunked"
		delete(resp.Headers, "Content-Length")
	} else {
		resp.Headers["Content-Length"] = strconv.Itoa(len(resp.Body))
		delete(resp.Headers, "Transfer-Encoding")
	}

	for name, value := range resp.Headers {
		headerLine := fmt.Sprintf("%s: %s\r\n", name, value)
		data = append(data, []byte(headerLine)...)
	}

	data = append(data, []byte("\r\n")...)

	if resp.Chunked {
		data = appendChunkedBody(data, resp.Body)
	} else {
		data = append(data, resp.Body...)
	}

	return data
}

func appendChunkedBody(data []byte, body []byte) []byte {
	const chunkSize = 8

	for len(body) > 0 {
		size := chunkSize
		if len(body) < size {
			size = len(body)
		}

		chunk := body[:size]

		data = append(data, []byte(fmt.Sprintf("%x\r\n", len(chunk)))...)
		data = append(data, chunk...)
		data = append(data, []byte("\r\n")...)

		body = body[size:]
	}

	data = append(data, []byte("0\r\n\r\n")...)

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
