//+build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/seaptc/server/data"
)

func main() {
	projectID := flag.String("project", "seaptc", "")
	flag.Parse()
	client, err := firestore.NewClient(context.Background(), *projectID)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	switch flag.Arg(0) {
	case "config-get":
		var config data.AppConfig
		if err := data.GetDocTo(ctx, client, data.AppConfigPath, &config); err != nil {
			log.Fatal(err)
		}
		p, _ := json.MarshalIndent(&config, "", "  ")
		fmt.Printf("%s\n", p)
	case "config-set":
		var config data.AppConfig
		if err := json.NewDecoder(os.Stdin).Decode(&config); err != nil {
			log.Fatal(err)
		}
		if _, err := client.Doc(data.AppConfigPath).Set(ctx, &config); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command %q", flag.Arg(0))
	}
}
