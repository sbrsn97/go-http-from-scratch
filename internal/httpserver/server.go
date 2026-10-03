package httpserver

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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

		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	reader := newRequestReader(conn)

	for {
		req, err := reader.readRequest()
		if err != nil {
			if errors.Is(err, io.EOF) {
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

		if err := writeResponse(conn, resp); err != nil {
			fmt.Println("write response error:", err)
			return
		}

		if shouldClose {
			return
		}
	}
}
