package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/atlanssia/fustgo/internal/config"
	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/logger"
)

var (
	version   = "0.1.0"
	configFile string
	showVersion bool
)

func init() {
	flag.StringVar(&configFile, "config", "configs/default.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("FustGo DataX version %s\n", version)
		os.Exit(0)
	}

	fmt.Println("=", "="*50)
	fmt.Println("  FustGo DataX - ETL/ELT Data Synchronization System")
	fmt.Printf("  Version: %s\n", version)
	fmt.Println("=", "="*50)

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.NewLogger(
		"fustgo",
		cfg.Observability.Logs.Local.Enabled,
		cfg.Observability.Logs.Local.Path+"/fustgo.log",
	)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()
	logger.SetDefaultLogger(log)

	log.Info("Starting FustGo DataX v%s", version)
	log.Info("Deployment mode: %s", cfg.Deployment.Mode)
	log.Info("Database type: %s", cfg.Database.Type)

	// Initialize database
	var metaStore database.MetadataStore
	switch cfg.Database.Type {
	case "sqlite":
		metaStore, err = database.NewSQLiteStore(cfg.Database.Path)
		if err != nil {
			log.Fatal("Failed to initialize SQLite store: %v", err)
		}
		log.Info("Using SQLite database: %s", cfg.Database.Path)
	default:
		log.Fatal("Unsupported database type: %s", cfg.Database.Type)
	}
	defer metaStore.Close()

	log.Info("Metadata store initialized successfully")

	// TODO: Start web server
	// TODO: Start worker pool
	// TODO: Start scheduler

	log.Info("FustGo DataX is ready")
	log.Info("Web UI available at: http://%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Keep running
	select {}
}