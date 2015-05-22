package main

import (
	// Stdlib
	"flag"
	"log"
	"os"

	// Vendor
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
}

func run() error {
	flagEnv := flag.String("env", "", "godotenv file to load into the environment")
	flag.Parse()

	if envFile := *flagEnv; envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return err
		}
	}

	server, err := LoadServerFromEnvironment()
	if err != nil {
		return err
	}

	server.Run()
	return nil
}
