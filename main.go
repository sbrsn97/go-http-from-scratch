package main

import (
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

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	fmt.Printf("read %d bytes\n", n)
	fmt.Println("-------raw data-------")
	fmt.Print(string(buffer[:n]))
	fmt.Println("----------------------")

	_, err = conn.Write([]byte("received\n"))
	if err != nil {
		fmt.Println("Write error:", err)
	}
}
