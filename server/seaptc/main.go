package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/seaptc/server/store"
)

func main() {
	isAppEngine := os.Getenv("GAE_INSTANCE") != ""
	if isAppEngine {
		log.SetFlags(0)
	}

	defaultAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		defaultAddr = ":" + port
	}

	addr := flag.String("addr", defaultAddr, "")
	dir := flag.String("dir", "assets", "")
	store.SetupFlags()
	flag.Parse()

	st, err := store.NewFromFlags(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	h, err := newApplication(context.Background(), st, !isAppEngine, *dir,
		&catalogService{},
		&dashboardService{},
		&loginService{},
		&participantService{},
		&apiService{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on addr %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
