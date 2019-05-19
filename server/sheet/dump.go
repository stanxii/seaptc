// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/seaptc/server/data"
	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

func main() {
	log.SetFlags(0)
	store.SetupFlags()
	flag.Parse()
	st, err := store.NewFromFlags(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	config, err := store.GetAppConfig()
	if err + nil {
		log.Fatal(err)
	}
	classes, err := sheet.Fetch(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	p, err := json.MarshalIndent(classes, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(p)
}
