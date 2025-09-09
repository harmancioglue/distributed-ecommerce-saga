package messaging

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type RabbitMQConfig struct {
	Host              string
	Port              int
	Username          string
	Password          string
	VHost             string
	Exchange          string
	RetryCount        int
	RetryDelay        time.Duration
	ConnectionTimeout time.Duration
}

func NewRabbitMQConfig() *RabbitMQConfig {
	port, _ := strconv.Atoi(getEnvOrDefault("RABBITMQ_PORT", "5672"))
	retryCount, _ := strconv.Atoi(getEnvOrDefault("RABBITMQ_RETRY_COUNT", "3"))

	return &RabbitMQConfig{
		Host:              getEnvOrDefault("RABBITMQ_HOST", "localhost"),
		Port:              port,
		Username:          getEnvOrDefault("RABBITMQ_USERNAME", "guest"),
		Password:          getEnvOrDefault("RABBITMQ_PASSWORD", "guest"),
		VHost:             getEnvOrDefault("RABBITMQ_VHOST", "/"),
		Exchange:          getEnvOrDefault("RABBITMQ_EXCHANGE", "saga.events"),
		RetryCount:        retryCount,
		RetryDelay:        time.Second * 5,
		ConnectionTimeout: time.Second * 30,
	}
}

func (c *RabbitMQConfig) ConnectionURL() string {
	vhost := c.VHost
	if vhost != "/" && !strings.HasPrefix(vhost, "/") {
		vhost = "/" + vhost
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.Username, c.Password, c.Host, c.Port, vhost)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
