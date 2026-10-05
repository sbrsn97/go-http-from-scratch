package httpserver

import (
	"os"
	"strings"
)

type Handler func(Request) Response
type Middleware func(Handler) Handler

func handleRequest(req Request) Response {
	if req.Path == "/" {
		if req.Method != "GET" {
			return methodNotAllowedResponse()
		}

		data, err := os.ReadFile("public/index.html")
		if err != nil {
			return internalServerErrorResponse()
		}

		return Response{
			StatusCode: 200,
			StatusText: "OK",
			Headers: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			Body: data,
		}
	}

	if strings.HasPrefix(req.Path, "/static/") {
		if req.Method != "GET" {
			return methodNotAllowedResponse()
		}

		return serveStaticFile(req.Path)
	}

	if req.Path == "/hello" {
		if req.Method != "GET" {
			return methodNotAllowedResponse()
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
			return methodNotAllowedResponse()
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

	return notFoundResponse()
}
