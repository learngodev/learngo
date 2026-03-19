package main

import (
	"log"

	"learn-go/internal/server"
)

func main() {
	application, err := server.New()
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("application terminated: %v", err)
	}
}
