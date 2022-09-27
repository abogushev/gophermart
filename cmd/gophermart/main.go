package main

import (
	"context"
	"gophermart/internal/config"
	"gophermart/internal/db"
	"gophermart/internal/processing"
	mainServer "gophermart/internal/server"
	"gophermart/internal/utils"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("error on create logger: %v", err)
	}
	logger := l.Sugar()
	defer logger.Sync()

	cnfg, err := config.NewConfig()
	if err != nil {
		logger.Fatalf("failed to parse config, %w", err)
	}
	logger.Infof("config is %v", cnfg)
	storage, err := db.NewStorage(cnfg.DBURL, ctx, logger)
	if err != nil {
		logger.Fatalf("failed to create storage, %w", err)
	}

	wg := &sync.WaitGroup{}

	processing.RunDaemon(http.Client{}, cnfg.ProcessingAddress, storage, logger, ctx, wg, cnfg)
	mainServer.Run(storage, utils.TestSecret, cnfg, logger, ctx)

	wg.Wait()
}
