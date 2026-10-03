package httpserver

import "os"

func handleRequest(req Request) Response {
	if req.Path == "/" {
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

		data, err := os.ReadFile("public/index.html")
		if err != nil {
			return Response{
				StatusCode: 500,
				StatusText: "Internal Server Error",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte("internal server error\n"),
			}
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
