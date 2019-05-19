//+build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/seaptc/server/data"
	"github.com/seaptc/server/store"
)

func main() {
	log.SetFlags(0)
	store.SetupFlags()
	flag.Parse()
	s, err := store.NewFromFlags(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	switch flag.Arg(0) {
	case "config-get":
		config, err := s.GetAppConfig(ctx)
		if err != nil {
			log.Fatal(err)
		}
		p, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("%s\n", p)
	case "config-set":
		var config data.AppConfig
		if err := json.NewDecoder(os.Stdin).Decode(&config); err != nil {
			log.Fatal(err)
		}
		if err := s.SetAppConfig(ctx, &config); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command %q, commands are config-get, config-set", flag.Arg(0))
	}
}
