package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)

func ListenAndServe(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	fmt.Println("Listening on:", addr)

	var wg sync.WaitGroup

	handler := Handler(handleRequest)
	if os.Getenv("HTTP_SERVER_LOGGING") == "1" {
		handler = loggingMiddleware(handler)
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutting down server...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}

			return err
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			handleConnection(conn, handler)
		}()
	}

	wg.Wait()
	return nil
}

func handleConnection(conn net.Conn, handler Handler) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	reader := newRequestReader(conn)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			fmt.Println("set read deadline error:", err)
			return
		}

		req, err := reader.readRequest()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				fmt.Println("connection timed out:", conn.RemoteAddr())
				return
			}

			switch {
			case errors.Is(err, errHeaderTooLarge):
				writeErrorAndClose(conn, requestHeaderTooLargeResponse())
			case errors.Is(err, errBodyTooLarge):
				writeErrorAndClose(conn, contentTooLargeResponse())
			default:
				writeErrorAndClose(conn, badRequestResponse())
			}

			return
		}

		fmt.Println("method:", req.Method)
		fmt.Println("path:", req.Path)

		shouldClose := strings.EqualFold(
			req.Headers["connection"],
			"close",
		)

		resp := handler(req)

		if shouldClose {
			if resp.Headers == nil {
				resp.Headers = make(map[string]string)
			}

			resp.Headers["Connection"] = "close"
		}

		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			fmt.Println("set write deadline error:", err)
			return
		}

		if err := writeResponse(conn, resp); err != nil {
			fmt.Println("write response error:", err)
			return
		}

		if shouldClose {
			return
		}
	}
}

func writeErrorAndClose(conn net.Conn, resp Response) {
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}

	resp.Headers["Connection"] = "close"

	if err := writeResponse(conn, resp); err != nil {
		fmt.Println("write response error:", err)
	}
}
