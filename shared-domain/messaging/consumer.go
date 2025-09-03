package messaging

import (
	"encoding/json"
	"fmt"
	"github.com/distributed-ecommerce-saga/shared-domain/events"
	"github.com/streadway/amqp"
	"log"
	"time"
)

type EventHandler func(event events.SagaEvent) error

type Consumer struct {
	client      *RabbitMQClient
	queueName   string
	serviceName string
}

func NewConsumer(client *RabbitMQClient, queueName, serviceName string) *Consumer {
	return &Consumer{
		client:      client,
		queueName:   queueName,
		serviceName: serviceName,
	}
}

func (c *Consumer) ConsumeEvents(routingKeys []string, handler EventHandler) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("There is no connection to RabbitMQ")
	}

	channel := c.client.Channel()

	queue, err := channel.QueueDeclare(
		c.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return fmt.Errorf("queue declare error: %v", err)
	}

	for _, routingKey := range routingKeys {
		err = channel.QueueBind(
			queue.Name,               // queue name
			routingKey,               // routing key
			c.client.config.Exchange, // exchange
			false,                    // no-wait
			nil,                      // arguments
		)
		if err != nil {
			return fmt.Errorf("queue bind error (%s): %v", routingKey, err)
		}
		log.Printf("Queue %s bound to routing key: %s", queue.Name, routingKey)
	}

	messages, err := channel.Consume(
		queue.Name,    // queue
		c.serviceName, // consumer
		false,         // auto-ack (manuel ack kullanacağız)
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return fmt.Errorf("consume start error: %v", err)
	}

	log.Printf("Consuming events on queue: %s", queue.Name)

	go func() {
		for {
			select {
			case msg := <-messages:
				c.handleMessage(msg, handler)
			case <-c.client.ctx.Done():
				log.Printf("Consumer is stooed: %s", c.serviceName)
				return
			}
		}
	}()

	return nil
}

func (c *Consumer) handleMessage(msg amqp.Delivery, handler EventHandler) {
	var event events.SagaEvent

	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("Event deserialize error: %v", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("Event received: %s from %s", event.EventType, event.Service)

	if err := handler(event); err != nil {
		log.Printf("Event process error: %v", err)

		if c.shouldRetry(msg) {
			c.republishWithRetry(msg, event)
		} else {
			log.Printf("Max retry is reached, dead letter sent to queue: %s", event.EventType)
			msg.Nack(false, false) // Dead letter queue
		}
		return
	}

	msg.Ack(false)
	log.Printf("Event processed successfully: %s", event.EventType)
}

func (c *Consumer) shouldRetry(msg amqp.Delivery) bool {
	if xDeath, ok := msg.Headers["x-death"]; ok {
		if deathArray, ok := xDeath.([]interface{}); ok && len(deathArray) > 0 {
			if death, ok := deathArray[0].(amqp.Table); ok {
				if count, ok := death["count"]; ok {
					if retryCount, ok := count.(int64); ok && retryCount >= 3 {
						return false
					}
				}
			}
		}
	}

	// first rety
	return true
}

func (c *Consumer) republishWithRetry(msg amqp.Delivery, event events.SagaEvent) {
	channel := c.client.Channel()

	time.Sleep(2 * time.Second)

	err := channel.Publish(
		msg.Exchange,
		msg.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			DeliveryMode: msg.DeliveryMode,
			Headers:      msg.Headers,
		},
	)

	if err != nil {
		log.Printf("Retry publish error: %v", err)
		msg.Nack(false, false)
		return
	}

	msg.Ack(false)
	log.Printf("Re-published: %s", event.EventType)
}
