package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/distributed-ecommerce-saga/payment-service/internal/gateway"
	"github.com/distributed-ecommerce-saga/payment-service/internal/handlers"
	"github.com/distributed-ecommerce-saga/payment-service/internal/repository"
	"github.com/distributed-ecommerce-saga/payment-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Payment Service starting...")

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
		log.Fatalf("RabbitMQ bağlantı hatası: %v", err)
	}
	defer rabbitClient.Close()

	failureRate := getEnvFloat("PAYMENT_FAILURE_RATE", 0.1) // 10% failure rate
	paymentGateway := gateway.NewMockPaymentGateway(failureRate)

	// Dependencies injection
	publisher := messaging.NewPublisher(rabbitClient)
	consumer := messaging.NewConsumer(rabbitClient, "payment-service-queue", "payment-service")

	paymentRepo := repository.NewPaymentRepository(db)
	paymentService := service.NewPaymentService(paymentRepo, paymentGateway, publisher)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	// Fiber app setup
	app := setupFiberApp()

	// Routes setup
	setupRoutes(app, paymentHandler)

	// RabbitMQ event consumption başlat
	go func() {
		log.Println("🐰 RabbitMQ event consumption starting...")
		if err := paymentHandler.StartConsuming(consumer); err != nil {
			log.Printf("RabbitMQ consumption error: %v", err)
		}
	}()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Payment Service closing...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	// Server starting
	port := getEnvOrDefault("PORT", "8002")
	log.Printf("🌍 Payment Service çalışıyor: http://localhost:%s", port)
	log.Printf("💳 Mock Payment Gateway aktif - Failure Rate: %.1f%%", failureRate*100)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server başlatma hatası: %v", err)
	}
}

func initDatabase() (*sql.DB, error) {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "payment_db")

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("database open error: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping error: %v", err)
	}

	log.Printf("✅ Database connection success: %s", dbName)
	return db, nil
}

func setupFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Payment Service v1.0",
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency} | IP: ${ip}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
	}))

	return app
}

func setupRoutes(app *fiber.App, paymentHandler *handlers.PaymentHandler) {
	// API v1 routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", paymentHandler.HealthCheck)

	// Payment routes
	orders := api.Group("/orders")
	orders.Get("/:order_id/payment", paymentHandler.GetPaymentByOrderID) // GET /api/v1/orders/:order_id/payment

	app.Use("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": "Route not found",
		})
	})
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	log.Printf("Error: %v", err)

	return c.Status(code).JSON(fiber.Map{
		"success":   false,
		"message":   message,
		"error":     err.Error(),
		"timestamp": fiber.Map{"error_time": "now"},
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
