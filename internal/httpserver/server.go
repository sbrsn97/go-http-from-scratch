package httpserver

import (
	"fmt"
	"net"
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

	req, err := readRequest(conn)
	if err != nil {
		fmt.Println("invalid request:", err)
		return
	}

	fmt.Println("method:", req.Method)
	fmt.Println("target:", req.Target)
	fmt.Println("path:", req.Path)
	fmt.Println("raw query:", req.RawQuery)
	fmt.Println("version:", req.Version)
	fmt.Println("headers:", req.Headers)
	fmt.Println("body:", string(req.Body))

	resp := handleRequest(req)

	err = writeResponse(conn, resp)
	if err != nil {
		fmt.Println("write response error:", err)
	}
}
