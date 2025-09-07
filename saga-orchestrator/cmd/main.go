package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/handlers"
	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/repository"
	"github.com/distributed-ecommerce-saga/saga-orchestrator/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Starting Saga Orchestrator...")

	// Database connection
	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	// RabbitMQ connection
	rabbitConfig := messaging.NewRabbitMQConfig()
	rabbitClient := messaging.NewRabbitMQClient(rabbitConfig)

	if err := rabbitClient.Connect(); err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer rabbitClient.Close()

	// Dependencies injection
	publisher := messaging.NewPublisher(rabbitClient)
	consumer := messaging.NewConsumer(rabbitClient, "saga-orchestrator-queue", "saga-orchestrator")

	sagaRepo := repository.NewSagaRepository(db)
	orchestrator := service.NewSagaOrchestrator(sagaRepo, publisher)
	eventHandler := handlers.NewEventHandler(orchestrator)

	// Start RabbitMQ event consumption
	go func() {
		log.Println("🐰 Starting RabbitMQ event consumption...")
		if err := eventHandler.StartConsuming(consumer); err != nil {
			log.Printf("RabbitMQ consumption error: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down Saga Orchestrator...")
	}()

	log.Println("✅ Saga Orchestrator is ready and listening for events")

	// Keep the application running
	select {}
}

func initDatabase() (*sql.DB, error) {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "orchestrator_db")

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("database open error: %v", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	// Connection test
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping error: %v", err)
	}

	log.Printf("✅ Database connection successful: %s", dbName)
	return db, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
