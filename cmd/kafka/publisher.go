package main

import (
	"WB-L0/internal/structs"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

func main() {
	producer, err := initializeProducer()
	if err != nil {
		log.Fatalf("Failed to initialize producer: %v", err)
	}
	defer closeProducer(producer)

	orders := generateOrders(10)
	for _, order := range orders {
		if err := publishOrder(producer, order); err != nil {
			log.Printf("Failed to publish order: %v", err)
		}
	}
}

func initializeProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	brokers := getBrokers()
	producer, err := connectToKafka(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("could not connect to Kafka: %w", err)
	}

	return producer, nil
}

func getBrokers() []string {
	brokersEnv := os.Getenv("KAFKA_BROKERS")
	if brokersEnv == "" {
		brokersEnv = "kafka:9092"
	}
	return strings.Split(brokersEnv, ",")
}

func connectToKafka(brokers []string, config *sarama.Config) (sarama.SyncProducer, error) {
	var producer sarama.SyncProducer
	var err error

	for attempt := 1; attempt <= 10; attempt++ {
		producer, err = sarama.NewSyncProducer(brokers, config)
		if err == nil {
			return producer, nil
		}
		log.Printf("Attempt %d: Failed to connect to Kafka: %v", attempt, err)
		time.Sleep(5 * time.Second)
	}

	return nil, err
}

func closeProducer(producer sarama.SyncProducer) {
	if err := producer.Close(); err != nil {
		log.Printf("Error closing producer: %v", err)
	}
}

func generateOrders(count int) []structs.Order {
	orders := make([]structs.Order, count)
	for i := 0; i < count; i++ {
		orders[i] = createOrder(i)
	}
	return orders
}

func createOrder(index int) structs.Order {
	return structs.Order{
		OrderUID:    fmt.Sprintf("b563feb7b2b84b6test%d", index),
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: structs.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: structs.Payment{
			Transaction:  fmt.Sprintf("b563feb7b2b84b6test%d", index),
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDT:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []structs.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
	}
}

func publishOrder(producer sarama.SyncProducer, order structs.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: "orders",
		Value: sarama.ByteEncoder(data),
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Order sent to partition %d with offset %d", partition, offset)
	return nil
}
