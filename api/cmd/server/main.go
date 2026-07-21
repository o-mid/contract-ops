package main

import (
	"log"
	"net/http"
	"os"

	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/httpapi"
)

func main() {
	address := os.Getenv("PORT")
	if address == "" {
		address = "8080"
	}

	server := httpapi.NewServer(events.NewStore(events.Fixtures()))
	log.Printf("contract-ops API listening on :%s", address)
	log.Fatal(http.ListenAndServe(":"+address, server.Handler()))
}
