package kafka

import (
	"WB-L0/internal/service"
	"WB-L0/internal/structs"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"WB-L0/internal/configs"
	"github.com/IBM/sarama"
)

type Consumer interface {
	Consume(wg *sync.WaitGroup, ctx context.Context) error
	Stop()
}

type consumer struct {
	ready   chan bool
	client  sarama.ConsumerGroup
	service service.Service
	config  configs.KafkaConfig
	groupID string
	topic   string
}

func NewConsumer(cfg configs.KafkaConfig, service service.Service) Consumer {
	return &consumer{
		ready:   make(chan bool),
		client:  createConsumerGroup(cfg),
		service: service,
		config:  cfg,
		groupID: cfg.GroupID,
		topic:   cfg.Topic,
	}
}

func createConsumerGroup(cfg configs.KafkaConfig) sarama.ConsumerGroup {
	config := sarama.NewConfig()
	config.Version = sarama.V2_7_0_0

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		log.Fatalf("Failed to create consumer group: %v", err)
	}
	return group
}

func (c *consumer) Consume(wg *sync.WaitGroup, ctx context.Context) error {
	for {
		if err := c.client.Consume(ctx, []string{c.topic}, c); err != nil {
			log.Printf("Consumer error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.ready = make(chan bool)
	}
}

func (c *consumer) Stop() {
	if err := c.client.Close(); err != nil {
		log.Printf("Error closing consumer client: %v", err)
	}
}

func (c *consumer) Setup(sarama.ConsumerGroupSession) error {
	close(c.ready)
	return nil
}

func (c *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("Received message: value = %s, timestamp = %v, topic = %s",
			string(msg.Value), msg.Timestamp, msg.Topic)

		if err := c.processMessage(msg); err != nil {
			log.Printf("Error processing message: %v", err)
			continue
		}

		session.MarkMessage(msg, "")
	}
	return nil
}

func (c *consumer) processMessage(msg *sarama.ConsumerMessage) error {
	var order structs.Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return err
	}

	return c.service.SaveOrder(context.Background(), order)
}
