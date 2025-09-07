package messaging

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/streadway/amqp"
)

type RabbitMQClient struct {
	config     *RabbitMQConfig
	connection *amqp.Connection
	channel    *amqp.Channel
	mu         sync.RWMutex
	isClosing  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewRabbitMQClient(config *RabbitMQConfig) *RabbitMQClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &RabbitMQClient{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Graceful shutdown için signal handling
	go client.handleGracefulShutdown()

	return client
}

func (r *RabbitMQClient) handleGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Signal received: %v. RabbitMQ connection is closing...", sig)
		r.Close()
	case <-r.ctx.Done():
		log.Println("Context cancelled. RabbitMQ is closing...")
		return
	}
}

func (r *RabbitMQClient) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	for i := 0; i < r.config.RetryCount; i++ {
		r.connection, err = amqp.Dial(r.config.ConnectionURL())
		if err != nil {
			log.Printf("RabbitMQ connection error (attempt %d/%d): %v", i+1, r.config.RetryCount, err)
			if i < r.config.RetryCount-1 {
				time.Sleep(r.config.RetryDelay)
				continue
			}
			return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
		}

		r.channel, err = r.connection.Channel()
		if err != nil {
			r.connection.Close()
			return fmt.Errorf("RabbitMQ channel açılamadı: %v", err)
		}

		err = r.channel.ExchangeDeclare(
			r.config.Exchange, // name
			"topic",           // type
			true,              // durable
			false,             // auto-deleted
			false,             // internal
			false,             // no-wait
			nil,               // arguments
		)
		if err != nil {
			r.channel.Close()
			r.connection.Close()
			return fmt.Errorf("failed to create exchange: %v", err)
		}

		log.Printf("Successfully connected to RabbitMQ: %s", r.config.Host)

		// Listen connection drops
		go r.handleReconnection()

		return nil
	}

	return err
}

func (r *RabbitMQClient) handleReconnection() {
	notifyClose := make(chan *amqp.Error)
	r.connection.NotifyClose(notifyClose)

	select {
	case err := <-notifyClose:
		if !r.isClosing {
			log.Printf("RabbitMQ connection is lost: %v. Trying reconnect...", err)
			time.Sleep(time.Second * 2)
			if reconnectErr := r.Connect(); reconnectErr != nil {
				log.Printf("Recconnect error: %v", reconnectErr)
			}
		}
	}
}

func (r *RabbitMQClient) Channel() *amqp.Channel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.channel
}

func (r *RabbitMQClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isClosing {
		return nil
	}

	r.isClosing = true
	r.cancel()

	var closeErr error

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			closeErr = fmt.Errorf("channel close error: %v", err)
			log.Printf("Failed to close channel: %v", err)
		}
	}

	if r.connection != nil {
		if err := r.connection.Close(); err != nil {
			if closeErr != nil {
				closeErr = fmt.Errorf("%v; connection close error: %v", closeErr, err)
			} else {
				closeErr = fmt.Errorf("connection close error: %v", err)
			}
			log.Printf("Failed to close connection: %v", err)
		}
	}

	if closeErr == nil {
		log.Println("RabbitMQ connection closed successfully")
	}

	return closeErr
}

func (r *RabbitMQClient) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.connection != nil && !r.connection.IsClosed()
}
