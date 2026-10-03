package httpserver

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
)

func ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	defer listener.Close()

	fmt.Println("Listening on:", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
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

			fmt.Println("read request error:", err)
			return
		}

		fmt.Println("method:", req.Method)
		fmt.Println("path:", req.Path)

		shouldClose := strings.EqualFold(
			req.Headers["connection"],
			"close",
		)

		resp := handleRequest(req)

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
