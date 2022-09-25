package main

import (
	"context"
	"gophermart/internal/config"
	"gophermart/internal/db"
	"gophermart/internal/processing"
	mainServer "gophermart/internal/server"
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
	logger.Debug(cnfg)
	storage, err := db.NewStorage(cnfg.DBURL, ctx, logger)
	if err != nil {
		logger.Fatalf("failed to create storage, %w", err)
	}

	wg := &sync.WaitGroup{}

	var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"
	processing.RunDaemon(http.Client{}, cnfg.ProcessingAddress, storage, logger, ctx, wg)
	mainServer.Run(storage, secret, cnfg, logger, ctx)

	wg.Wait()
}
