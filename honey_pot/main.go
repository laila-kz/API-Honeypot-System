package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"honeypot-go/config"
	"honeypot-go/handlers"
)

func main() {
	log.Println("Starting API Honeypot System...")
	log.Printf("Config: addr=%s ml=%s db=%s timeout=%s",
		config.ListenAddr(),
		config.AppConfig.MLServiceURL,
		config.AppConfig.LogDBPath,
		config.AppConfig.RequestTimeout,
	)

	handler, err := handlers.NewAPIHandler()
	if err != nil {
		log.Fatalf("Failed to initialize handler: %v", err)
	}
	defer handler.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/admin", handler.Admin)
	mux.HandleFunc("/api/v1/users", handler.Users)
	mux.HandleFunc("/api/v1/auth", handler.Auth)
	mux.HandleFunc("/reset-password", handler.ResetPassword)
	mux.HandleFunc("/dashboard", handler.Dashboard)
	mux.HandleFunc("/logs", handler.ViewLogs)
	mux.HandleFunc("/", handler.NotFound)

	server := &http.Server{
		Addr:         config.ListenAddr(),
		Handler:      mux,
		ReadTimeout:  config.AppConfig.RequestTimeout,
		WriteTimeout: config.AppConfig.RequestTimeout + config.AppConfig.AttackDelay + 2*time.Second,
	}

	go func() {
		log.Printf("Honeypot server listening on %s", config.ListenAddr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down honeypot server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown error: %v", err)
	}
	log.Printf("Logs saved to %s / %s", config.AppConfig.LogDBPath, config.AppConfig.LogJSONPath)
}
