package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/seaptc/server/datastore"
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
	datastore.SetupFlags()
	flag.Parse()

	dsClient, err := datastore.NewFromFlags(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	h, err := newApplication(context.Background(), dsClient, !isAppEngine, *dir,
		&catalogService{},
		&loginService{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on addr %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, h))
}
