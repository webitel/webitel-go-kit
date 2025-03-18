package main

import (
	"flag"
	"log"
	"os"

	"github.com/webitel/webitel-go-kit/cmd/jet-gen-webitel/config"
	"github.com/webitel/webitel-go-kit/cmd/jet-gen-webitel/gen"
)

var configFile = flag.String("config", ".jet-gen.yaml", "path to config file")

func main() {
	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := config.Parse(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := gen.Generate(*cfg); err != nil {
		log.Fatal(err)
	}
}
