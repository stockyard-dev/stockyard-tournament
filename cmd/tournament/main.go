package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-tournament/internal/server"
	"github.com/stockyard-dev/stockyard-tournament/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9804"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./tournament-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("tournament: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Tournament — Self-hosted tournament brackets and event management\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n", port, port)
	log.Printf("tournament: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
