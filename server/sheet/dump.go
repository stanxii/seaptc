// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/seaptc/server/sheet"
	"github.com/seaptc/server/store"
)

func main() {
	log.SetFlags(0)
	store.SetupFlags()
	flag.Parse()
	ctx := context.Background()

	st, err := store.NewFromFlags(ctx)
	if err != nil {
		log.Fatal(err)
	}
	config, err := st.GetAppConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	classes, err := sheet.GetClasses(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	p, err := json.MarshalIndent(classes, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(p)
}
