package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"honeypot-go/config"
	"honeypot-go/handlers"
)

func main() {
	log.Println("Starting API Honeypot System...")

	handler, err := handlers.NewAPIHandler()
	if err != nil {
		log.Fatalf("Failed to initialize handler: %v", err)
	}
	defer handler.Close()

	// Setup routes
	http.HandleFunc("/login", handler.Login)
	http.HandleFunc("/admin", handler.Admin)
	http.HandleFunc("/api/v1/users", handler.Users)
	http.HandleFunc("/api/v1/auth", handler.Auth)
	http.HandleFunc("/reset-password", handler.ResetPassword)
	http.HandleFunc("/dashboard", handler.Dashboard)
	http.HandleFunc("/logs", handler.ViewLogs) // Bonus endpoint

	// Add a catch-all for other paths (404 with fake data)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Endpoint not found", "message": "API version deprecated"}`))
	})

	// Start server
	server := &http.Server{
		Addr:         config.AppConfig.ServerPort,
		ReadTimeout:  config.AppConfig.RequestTimeout,
		WriteTimeout: config.AppConfig.RequestTimeout,
	}

	go func() {
		log.Printf("Honeypot server listening on %s", config.AppConfig.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down honeypot server...")
	log.Println("Logs saved to ./logs/")
}
