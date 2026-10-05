package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/sbrsn97/go-http-from-scratch/internal/httpserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := httpserver.ListenAndServe(ctx, ":8080")
	if err != nil {
		fmt.Println("server error:", err)
	}
}
