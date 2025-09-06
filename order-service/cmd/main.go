package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/distributed-ecommerce-saga/order-service/internal/handlers"
	"github.com/distributed-ecommerce-saga/order-service/internal/repository"
	"github.com/distributed-ecommerce-saga/order-service/internal/service"
	"github.com/distributed-ecommerce-saga/shared-domain/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Order Service starting...")

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

	// Dependencies injection
	publisher := messaging.NewPublisher(rabbitClient)
	consumer := messaging.NewConsumer(rabbitClient, "order-service-queue", "order-service")

	orderRepo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(orderRepo, publisher)
	orderHandler := handlers.NewOrderHandler(orderService)

	// Fiber app setup
	app := setupFiberApp()

	// Routes setup
	setupRoutes(app, orderHandler)

	// RabbitMQ event consumption start
	go func() {
		if err := orderHandler.StartConsuming(consumer); err != nil {
			log.Printf("RabbitMQ consumption error: %v", err)
		}
	}()

	// Graceful shutdown setup
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Order Service closing...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	port := getEnvOrDefault("PORT", "8001")
	log.Printf("🌍 Order Service working: http://localhost:%s", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server start error hatası: %v", err)
	}
}

func initDatabase() (*sql.DB, error) {
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "order_db")

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("database open hatası: %v", err)
	}

	// Connection test
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping hatası: %v", err)
	}

	log.Printf("✅ Database bağlantısı başarılı: %s", dbName)
	return db, nil
}

func setupFiberApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Order Service v1.0",
		ErrorHandler: errorHandler,
	})

	// Middlewares
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

func setupRoutes(app *fiber.App, orderHandler *handlers.OrderHandler) {
	// API v1 routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", orderHandler.HealthCheck)

	// Order routes
	orders := api.Group("/orders")
	orders.Post("/", orderHandler.CreateOrder)    // POST /api/v1/orders
	orders.Get("/:id", orderHandler.GetOrderByID) // GET /api/v1/orders/:id

	// Customer routes
	customers := api.Group("/customers")
	customers.Get("/:customer_id/orders", orderHandler.GetOrdersByCustomerID) // GET /api/v1/customers/:customer_id/orders

	// Route not found
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
		"timestamp": fiber.Map{"error": err.Error()},
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
