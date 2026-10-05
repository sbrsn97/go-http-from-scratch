package httpserver

import (
	"fmt"
	"time"
)

func loggingMiddleware(next Handler) Handler {
	return func(req Request) Response {
		start := time.Now()

		resp := next(req)

		duration := time.Since(start)

		fmt.Printf(
			"%s %s %d %s\n",
			req.Method,
			req.Path,
			resp.StatusCode,
			duration,
		)

		return resp
	}
}
