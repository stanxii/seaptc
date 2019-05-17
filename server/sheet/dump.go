// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/seaptc/server/data"
	"github.com/seaptc/server/datastore"
	"github.com/seaptc/server/sheet"
)

func main() {
	log.SetFlags(0)
	datastore.SetupFlags()
	flag.Parse()
	ds, err := datastore.NewFromFlags(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	var config data.AppConfig
	if err := ds.GetDocTo(context.Background(), data.AppConfigPath, &config); err != nil {
		log.Fatal(err)
	}
	classes, err := sheet.Fetch(context.Background(), &config)
	if err != nil {
		log.Fatal(err)
	}
	p, err := json.MarshalIndent(classes, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(p)
}
