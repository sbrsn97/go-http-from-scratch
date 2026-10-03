package main

import (
	"fmt"
	"github.com/sbrsn97/go-http-from-scratch/internal/httpserver"
)

func main() {
	err := httpserver.ListenAndServe(":8080")
	if err != nil {
		fmt.Println("server error:", err)
	}
}
