package main

import (
	"./libs"
	"flag"
	"log"
)

func handleFlags() *daemon.Config {
	config := &daemon.Config{}

	flag.StringVar(&config.Listen, "listen", ":8000", "HTTP listen")

	flag.Parse()
	return config
}

func main() {
	config := handleFlags()

	if err := daemon.Run(config); err != nil {
		log.Printf("Daemon error %v:", err)
	}
}
