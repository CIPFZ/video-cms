package main

import (
	"context"
	"flag"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/core/db"
	"github.com/CIPFZ/video-cms/internal/migrations"
	"go.uber.org/zap"
	"log"
	"time"
)

const defaultConfigPath = "./configs/config.yaml"

func main() {
	path := flag.String("f", defaultConfigPath, "config file path")
	flag.Parse()
	cfg, _, err := config.Load(*path)
	if err != nil {
		log.Fatal(err)
	}
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	database, err := db.InitDatabase(cfg.Database, logger)
	if err != nil {
		log.Fatal(err)
	}
	pool, err := database.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := migrations.Run(ctx, database, cfg, logger); err != nil {
		log.Fatal(err)
	}
	log.Print("Migrations complete: ", migrations.Latest)
}
