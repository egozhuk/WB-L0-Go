package main

import (
	kafkaGateway "WB-L0/internal/gateways/kafka"
	"WB-L0/internal/repository/postgres"
	"WB-L0/internal/service"
	"context"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"WB-L0/internal/configs"
	httpGateway "WB-L0/internal/gateways/http"
	"WB-L0/internal/repository"
)

func main() {
	config, err := loadConfiguration()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	database, err := connectToDatabase(config)
	if err != nil {
		fmt.Printf("Failed to connect to the database: %v\n", err)
		return
	}
	defer closeDatabaseConnection(database)

	repositories := repository.NewRepository(database)
	services := service.NewService(repositories)
	if err := services.RestoreCache(); err != nil {
		fmt.Printf("Failed to restore cache from the database: %v\n", err)
		return
	}

	kafkaConsumer, err := initializeKafkaConsumer(config, services)
	if err != nil {
		fmt.Printf("Kafka is unavailable: %v\n", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	var waitGroup sync.WaitGroup

	startKafkaConsumer(kafkaConsumer, &waitGroup, ctx)

	httpServer := setupHTTPServer(config, services)
	startHTTPServer(httpServer)

	waitForShutdownSignal(cancel, &waitGroup, kafkaConsumer, httpServer)
}

func loadConfiguration() (configs.Config, error) {
	absolutePath, err := filepath.Abs(".")
	if err != nil {
		fmt.Printf("Failed to get absolute path: %v\n", err)
		return configs.Config{}, err
	}
	fmt.Printf("Absolute path: %s\n", absolutePath)

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "internal/configs/config.yaml"
	}

	return configs.LoadConfig(configPath)
}

func connectToDatabase(config configs.Config) (*sqlx.DB, error) {
	fmt.Printf("Connecting to database at %s:%s\n", config.DataBase.Host, config.DataBase.Port)

	db, err := postgres.NewPostgresDB(configs.DBConfig{
		Host:     config.DataBase.Host,
		Port:     config.DataBase.Port,
		Username: config.DataBase.Username,
		DBName:   config.DataBase.DBName,
		SSLMode:  config.DataBase.SSLMode,
		Password: config.DataBase.Password,
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func closeDatabaseConnection(db *sqlx.DB) {
	if err := db.Close(); err != nil {
		fmt.Printf("Error closing database: %v\n", err)
	}
}

func initializeKafkaConsumer(config configs.Config, services service.Service) (kafkaGateway.Consumer, error) {
	time.Sleep(40 * time.Second)
	kafkaClient := kafkaGateway.NewConsumer(config.Kafka, services)

	if err := waitForKafkaAvailability(config.Kafka.Brokers, 100*time.Second); err != nil {
		return nil, err
	}

	fmt.Printf("Subscribed to Kafka topic: %s\n", config.Kafka.Topic)
	return kafkaClient, nil
}

func startKafkaConsumer(kafkaClient kafkaGateway.Consumer, wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := kafkaClient.Consume(wg, ctx); err != nil {
			fmt.Printf("Failed to subscribe to topic: %v\n", err)
		}
	}()
}

func setupHTTPServer(config configs.Config, services service.Service) *httpGateway.Server {
	host := config.Server.Host
	port, err := strconv.Atoi(config.Server.Port)
	if err != nil {
		fmt.Printf("Invalid server port, defaulting to 8080: %v\n", err)
		port = 8080
	}

	return httpGateway.NewServer(services, host, uint16(port))
}

func startHTTPServer(server *httpGateway.Server) {
	go func() {
		fmt.Printf("HTTP server running on %s:%d\n", server.Host, server.Port)
		if err := server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Error starting HTTP server: %v\n", err)
		}
	}()
}

func waitForShutdownSignal(cancel context.CancelFunc, wg *sync.WaitGroup, kafkaClient kafkaGateway.Consumer, server *httpGateway.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	cancel()
	wg.Wait()

	kafkaClient.Stop()

	downCtx, downCanc := context.WithTimeout(context.Background(), 20*time.Second)
	defer downCanc()
	if err := server.Shutdown(downCtx); err != nil {
		fmt.Printf("Error shutting down HTTP server: %v\n", err)
	}

	fmt.Println("Server successfully stopped")
}

func waitForKafkaAvailability(brokers []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, broker := range brokers {
			conn, err := net.DialTimeout("tcp", broker, 4*time.Second)
			if err != nil {
				fmt.Printf("Failed to connect to Kafka broker %s: %v\n", broker, err)
				break
			}
			_ = conn.Close()
		}
		return nil
	}
	return errors.New("Kafka is unavailable after waiting")
}
