//+build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/seaptc/server/model"
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
		var config model.AppConfig
		if err := json.NewDecoder(os.Stdin).Decode(&config); err != nil {
			log.Fatal(err)
		}
		if err := s.SetAppConfig(ctx, &config); err != nil {
			log.Fatal(err)
		}
	case "conf-get":
		conf, err := s.GetConference(ctx)
		if err != nil {
			log.Fatal(err)
		}
		p, _ := json.MarshalIndent(conf, "", "  ")
		fmt.Printf("%s\n", p)
	case "conf-set":
		var conf model.Conference
		if err := json.NewDecoder(os.Stdin).Decode(&conf); err != nil {
			log.Fatal(err)
		}
		if err := s.SetConference(ctx, &conf); err != nil {
			log.Fatal(err)
		}
	case "update-classes":
		if err := s.UpdateClasses(ctx); err != nil {
			log.Fatal(err)
		}
	case "update-participants":
		if err := s.UpdateParticipants(ctx); err != nil {
			log.Fatal(err)
		}
	case "participant-get":
		p, err := s.GetParticipant(ctx, flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		b, _ := json.MarshalIndent(p, "", "  ")
		fmt.Printf("%s\n", b)
	case "participant-set":
		var p model.Participant
		if err := json.NewDecoder(os.Stdin).Decode(&p); err != nil {
			log.Fatal(err)
		}
		if err := s.DebugSetParticipant(ctx, &p); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Unknown command %q, commands are config-get, config-set", flag.Arg(0))
	}
}
