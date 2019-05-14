// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/seaptc/server/data"
	"github.com/seaptc/server/sheet"
)

func main() {
	log.SetFlags(0)
	projectID := flag.String("project", "seaptc", "")
	flag.Parse()
	fs, err := firestore.NewClient(context.Background(), *projectID)
	if err != nil {
		log.Fatal(err)
	}
	var config data.AppConfig
	if err := data.GetDocTo(context.Background(), fs, data.AppConfigPath, &config); err != nil {
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
