module github.com/distributed-ecommerce-saga/payment-service

go 1.24

toolchain go1.24.5

require (
	github.com/distributed-ecommerce-saga/shared-domain v1.0.0
	github.com/gofiber/fiber/v2 v2.51.0
	github.com/google/uuid v1.4.0
	github.com/lib/pq v1.10.9
	github.com/streadway/amqp v1.1.0
)

replace github.com/distributed-ecommerce-saga/shared-domain => ../shared-domain
