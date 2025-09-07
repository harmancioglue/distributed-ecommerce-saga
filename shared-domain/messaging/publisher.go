package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type Publisher struct {
	client *RabbitMQClient
}

func NewPublisher(client *RabbitMQClient) *Publisher {
	return &Publisher{
		client: client,
	}
}

func (p *Publisher) PublishSagaEvent(event events.SagaEvent) error {
	if !p.client.IsConnected() {
		return fmt.Errorf("There is no connection to RabbitMQ")
	}

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("event serialization error: %v", err)
	}

	routingKey := fmt.Sprintf("saga.%s.%s", event.Service, string(event.EventType))

	channel := p.client.Channel()
	err = channel.Publish(
		p.client.config.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Message persistence
			MessageId:    event.ID.String(),
			Timestamp:    event.Timestamp,
			Headers: amqp.Table{
				"saga_id":        event.SagaID.String(),
				"order_id":       event.OrderID.String(),
				"correlation_id": event.CorrelationID.String(),
				"service":        event.Service,
				"event_type":     string(event.EventType),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("event publish error: %v", err)
	}

	log.Printf("Event published: %s -> %s", routingKey, event.EventType)
	return nil
}

func (p *Publisher) PublishWithRetry(event events.SagaEvent, maxRetries int) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		if err := p.PublishSagaEvent(event); err != nil {
			lastErr = err
			log.Printf("Publish error (retry %d/%d): %v", i+1, maxRetries, err)

			if i < maxRetries-1 {
				time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
				continue
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("event publish başarısız (%d deneme): %v", maxRetries, lastErr)
}
