package main

import (
	"bytes"
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening on: 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	readBuffer := make([]byte, 1024)
	var data []byte

	for {
		n, err := conn.Read(readBuffer)
		if err != nil {
			fmt.Println("Read error:", err)
			return
		}

		data = append(data, readBuffer[:n]...)

		fmt.Printf("read %d bytes, total %d bytes\n", n, len(data))

		if bytes.Contains(data, []byte("\r\n\r\n")) {
			break
		}
	}

	fmt.Println("------- complete header block -------")
	fmt.Print(string(data))
	fmt.Println("-------------------------------------")

	_, err := conn.Write([]byte("received\n"))
	if err != nil {
		fmt.Println("Write error:", err)
	}
}
