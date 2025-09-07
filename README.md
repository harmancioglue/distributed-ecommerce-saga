# Distributed E-commerce Saga Pattern

A comprehensive implementation of the **Saga Pattern** for distributed transactions in a microservices e-commerce system.

## 🎯 Architecture Overview

This project demonstrates a **choreography-based saga pattern** with **centralized orchestration** for managing distributed transactions across multiple microservices.

### 🏗️ System Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Order Service │    │ Saga Orchestrator│    │ Payment Service │
│    Port: 8001   │◄──►│   (Internal)     │◄──►│   Port: 8002    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│Inventory Service│    │    RabbitMQ      │    │Shipping Service │
│   Port: 8003    │◄──►│  Message Broker  │◄──►│   Port: 8004    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                        │
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│Notification Svc │    │   PostgreSQL     │    │  Shared Domain  │
│   Port: 8005    │    │    Database      │    │    Library      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### 🔄 Saga Flow

**Happy Path (Success):**
```
Order Created → Payment Processed → Inventory Reserved → Shipping Created → Notification Sent → COMPLETED
```

**Compensation Path (Failure):**
```
Order Cancelled ← Payment Refunded ← Inventory Released ← Shipping Cancelled ← COMPENSATED
```

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for local development)
- PostgreSQL 15+ (if running locally)
- RabbitMQ 3.12+ (if running locally)

### 🐳 Using Docker Compose (Recommended)

1. **Clone and start all services:**
```bash
git clone <repository-url>
cd distributed-ecommerce-saga

# Start all services with one command
docker-compose up --build
```

2. **Verify all services are running:**
```bash
# Check service health
curl http://localhost:8001/api/v1/health  # Order Service
curl http://localhost:8002/api/v1/health  # Payment Service
curl http://localhost:8003/api/v1/health  # Inventory Service
curl http://localhost:8004/api/v1/health  # Shipping Service
curl http://localhost:8005/api/v1/health  # Notification Service

# Access RabbitMQ Management UI
open http://localhost:15672  # saga_user / saga_password
```

3. **Create a test order:**
```bash
curl -X POST http://localhost:8001/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "123e4567-e89b-12d3-a456-426614174000",
    "items": [
      {
        "product_id": "550e8400-e29b-41d4-a716-446655440001",
        "quantity": 2,
        "price": 1299.99
      }
    ],
    "shipping_address": {
      "street": "123 Main St",
      "city": "New York", 
      "state": "NY",
      "zip_code": "10001",
      "country": "USA"
    }
  }'
```

## 📋 API Endpoints

### Order Service (Port 8001)
- `POST /api/v1/orders` - Create new order
- `GET /api/v1/orders/:id` - Get order details
- `GET /api/v1/customers/:customer_id/orders` - Get customer orders

### Payment Service (Port 8002)
- `GET /api/v1/orders/:order_id/payment` - Get payment details

### Inventory Service (Port 8003)
- `GET /api/v1/health` - Health check

### Shipping Service (Port 8004)
- `GET /api/v1/orders/:order_id/shipment` - Get shipment details

### Notification Service (Port 8005)
- `GET /api/v1/health` - Health check

## 🔧 Configuration

### Environment Variables

Each service can be configured using environment variables:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=saga_user
DB_PASSWORD=saga_password
DB_NAME=service_specific_db

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USERNAME=saga_user
RABBITMQ_PASSWORD=saga_password
RABBITMQ_VHOST=saga_vhost

# Service Specific
PAYMENT_FAILURE_RATE=0.1      # 10% payment failure rate
SHIPPING_FAILURE_RATE=0.05    # 5% shipping failure rate
NOTIFICATION_FAILURE_RATE=0.02 # 2% notification failure rate
```

## 🧪 Testing Scenarios

### 1. Successful Order Flow
```bash
# 1. Create order
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8001/api/v1/orders -H "Content-Type: application/json" -d '{...}')
ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.data.id')

# 2. Watch saga progression (check logs)
docker-compose logs -f saga-orchestrator

# 3. Verify final order status
curl http://localhost:8001/api/v1/orders/$ORDER_ID
```

### 2. Payment Failure Scenario
```bash
# Payment service has 10% failure rate by default
# Keep creating orders until one fails to see compensation flow

# Watch compensation in action
docker-compose logs -f saga-orchestrator payment-service
```

### 3. Inventory Shortage
```bash
# Create order with high quantity to trigger inventory failure
curl -X POST http://localhost:8001/api/v1/orders -H "Content-Type: application/json" -d '{
  "customer_id": "123e4567-e89b-12d3-a456-426614174000",
  "items": [
    {
      "product_id": "550e8400-e29b-41d4-a716-446655440001", 
      "quantity": 1000,  # Higher than available stock
      "price": 1299.99
    }
  ],
  ...
}'
```

## 📊 Monitoring

### Service Logs
```bash
# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f order-service
docker-compose logs -f saga-orchestrator
```

### RabbitMQ Management
- **URL**: http://localhost:15672
- **Username**: saga_user
- **Password**: saga_password

### Database Access
```bash
# Connect to PostgreSQL
docker exec -it saga-postgres psql -U saga_user -d saga_main

# List all databases
\l

# Connect to specific service database
\c order_db
\dt  # List tables
```

## 🏗️ Development

### Local Development Setup
```bash
# 1. Start infrastructure only
docker-compose up postgres rabbitmq

# 2. Run services locally
cd order-service && go run cmd/main.go
cd payment-service && go run cmd/main.go
# ... etc
```

### Project Structure
```
distributed-ecommerce-saga/
├── shared-domain/           # Common types, messaging, HTTP utils
├── saga-orchestrator/       # Central saga coordinator
├── order-service/          # Order management
├── payment-service/        # Payment processing
├── inventory-service/      # Stock management
├── shipping-service/       # Shipment handling
├── notification-service/   # Customer notifications
├── scripts/               # Database initialization
├── docker-compose.yml     # Full stack setup
└── README.md             # This file
```

## 🧰 Technologies Used

- **Backend**: Go 1.21+, Fiber Web Framework
- **Database**: PostgreSQL 15 with JSONB
- **Message Broker**: RabbitMQ 3.12
- **Containerization**: Docker & Docker Compose
- **Patterns**: Saga Pattern, CQRS, Event Sourcing, Repository Pattern

## 🎭 Design Patterns

1. **Saga Pattern**: Distributed transaction management
2. **CQRS**: Command Query Responsibility Segregation  
3. **Event Sourcing**: Event-driven architecture
4. **Repository Pattern**: Data access abstraction
5. **Domain-Driven Design**: Business logic encapsulation

## 🚨 Error Handling

### Saga Compensation
The system automatically handles failures through compensation transactions:

1. **Payment Failure** → No compensation needed (order cancelled)
2. **Inventory Failure** → Refund payment
3. **Shipping Failure** → Release inventory + Refund payment  
4. **Notification Failure** → Non-critical, logged but doesn't trigger compensation

### Retry Mechanism
- RabbitMQ messages have built-in retry with exponential backoff
- Failed messages go to dead letter queues after max retries
- Manual intervention available through RabbitMQ management UI

## 🔐 Security Considerations

- Database credentials use environment variables
- RabbitMQ virtual hosts isolate message traffic
- Services communicate only through defined interfaces
- Input validation on all API endpoints

## 📈 Scalability

### Horizontal Scaling
```yaml
# Scale specific services
docker-compose up --scale payment-service=3
docker-compose up --scale inventory-service=2
```

### Database Sharding
Each service has its own database following the **Database per Service** pattern.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Saga Pattern implementation inspired by microservices best practices
- Event-driven architecture patterns
- Distributed systems design principles

---

**Happy Coding!** 🚀

For questions or support, please open an issue in the GitHub repository.
