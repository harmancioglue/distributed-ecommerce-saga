module github.com/distributed-ecommerce-saga/saga-orchestrator

go 1.21


require (
	github.com/distributed-ecommerce-saga/shared-domain v1.0.0
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
)

require github.com/streadway/amqp v1.1.0 // indirect

replace github.com/distributed-ecommerce-saga/shared-domain => ./shared-domain
