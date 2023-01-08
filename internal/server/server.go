package server

import (
	"context"
	"errors"
	"gophermart/internal/account"
	"gophermart/internal/auth"
	"gophermart/internal/db"
	"gophermart/internal/order"
	"gophermart/internal/withdrawals"
	"net/http"

	"gophermart/internal/config"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
)

func Run(db db.Storage, authSecret string, cfg *config.Config, logger *zap.SugaredLogger, ctx context.Context) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	authHandler := auth.NewHandler(db, authSecret, logger)
	orderHandler := order.NewHandler(db, authSecret, logger)
	accountHandler := account.NewAccountHandler(db, authSecret, logger)
	withdrawalsHandler := withdrawals.NewHandler(db, authSecret)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Auth)
		r.Post("/orders", orderHandler.PostOrder)
		r.Get("/orders", orderHandler.GetOrders)
		r.Get("/balance", accountHandler.GetAccount)
		r.Post("/balance/withdraw", accountHandler.PostWithdraw)
		r.Get("/withdrawals", withdrawalsHandler.GetWithdrawals)
	})

	server := &http.Server{Addr: cfg.Address, Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("server start error: %w", err)
		}
	}()
	logger.Info("server started successfuly")

	<-ctx.Done()
	logger.Info("get stop signal, start shutdown server")
	if err := server.Shutdown(ctx); err != nil && errors.Is(err, context.Canceled) {
		logger.Fatalf("Server Shutdown Failed:%w", err)
	} else {
		logger.Info("server stopped successfully")
	}
}
