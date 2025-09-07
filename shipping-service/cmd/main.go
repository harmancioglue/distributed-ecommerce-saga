package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/handlers"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/repository"
	"github.com/distributed-ecommerce-saga/shipping-service/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Starting Shipping Service...")

	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	rabbitConfig := messaging.NewRabbitMQConfig()
	rabbitClient := messaging.NewRabbitMQClient(rabbitConfig)

	if err := rabbitClient.Connect(); err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer rabbitClient.Close()

	failureRate := getEnvFloat("SHIPPING_FAILURE_RATE", 0.05)

	publisher := messaging.NewPublisher(rabbitClient)
	consumer := messaging.NewConsumer(rabbitClient, "shipping-service-queue", "shipping-service")

	shippingRepo := repository.NewShippingRepository(db)
	shippingService := service.NewShippingService(shippingRepo, publisher, failureRate)
	shippingHandler := handlers.NewShippingHandler(shippingService)

	app := setupFiberApp()
	setupRoutes(app, shippingHandler)

	go func() {
		log.Println("🐰 Starting RabbitMQ event consumption...")
		if err := shippingHandler.StartConsuming(consumer); err != nil {
			log.Printf("RabbitMQ consumption error: %v", err)
		}
	}()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down Shipping Service...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	port := getEnvOrDefault("PORT", "8004")
	log.Printf("🌍 Shipping Service running on: http://localhost:%s", port)
	log.Printf("📦 Mock Shipping Provider active - Failure Rate: %.1f%%", failureRate*100)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}

func initDatabase() (*sql.DB, error) {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "shipping_db")

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

	log.Printf("✅ Database connection successful: %s", dbName)
	return db, nil
}

func setupFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Shipping Service v1.0",
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Request-ID",
	}))

	return app
}

func setupRoutes(app *fiber.App, shippingHandler *handlers.ShippingHandler) {
	api := app.Group("/api/v1")
	api.Get("/health", shippingHandler.HealthCheck)

	orders := api.Group("/orders")
	orders.Get("/:order_id/shipment", shippingHandler.GetShipmentByOrderID)

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
		"timestamp": "now",
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
