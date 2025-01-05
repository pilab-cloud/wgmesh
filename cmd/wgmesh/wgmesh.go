package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/pilab-cloud/wgmesh"
)

var (
	Version     = "dev"
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("wgmesh version %s\n", Version)
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		println("Usage: wgmesh [config_file]")
		os.Exit(1)
	}

	configFile := os.Args[1]

	mesh, err := wgmesh.NewWgMesh(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create wgmesh")
		os.Exit(1)
	}

	go func() {
		err := mesh.Start()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start wgmesh")
			os.Exit(1)
		}
	}()

	// Wait for SIGINT or SIGTERM
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}
