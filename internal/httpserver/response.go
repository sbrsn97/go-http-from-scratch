package httpserver

import (
	"fmt"
	"net"
	"strconv"
)

type Response struct {
	StatusCode int
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
		statusText(resp.StatusCode),
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

func textResponse(statusCode int, body string) Response {
	return Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "text/plain; charset=utf-8",
		},
		Body: []byte(body),
	}
}

func notFoundResponse() Response {
	return textResponse(
		404,
		"404 Not Found\n",
	)
}

func methodNotAllowedResponse() Response {
	return textResponse(
		405,
		"method not allowed\n",
	)
}

func internalServerErrorResponse() Response {
	return textResponse(
		500,
		"internal server error\n",
	)
}

func badRequestResponse() Response {
	return textResponse(
		400,
		"400 Bad Request\n",
	)
}

func requestHeaderTooLargeResponse() Response {
	return textResponse(
		431,
		"request headers too large\n",
	)
}

func contentTooLargeResponse() Response {
	return textResponse(
		413,
		"request body too large\n",
	)
}

func statusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 413:
		return "Content Too Large"
	case 431:
		return "Request Header Fields Too Large"
	case 500:
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}
