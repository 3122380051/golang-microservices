package main

import (
	"fmt"
	"os"

	"github.com/3122380051/golang-microservices/internal/config"
	"github.com/3122380051/golang-microservices/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate {up|down|status}")
		os.Exit(1)
	}

	action := os.Args[1]
	if err := database.Migrate(cfg.DatabaseURL, "migrations", action); err != nil {
		fmt.Fprintf(os.Stderr, "migration error: %v\n", err)
		os.Exit(1)
	}
}
