package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

// TransactionEvent represents a transaction event received from NATS
type TransactionEvent struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Value       string    `json:"value"`
	Data        string    `json:"data"`
	BlockNumber string    `json:"block_number"`
	BlockHash   string    `json:"block_hash"`
	Timestamp   time.Time `json:"timestamp"`
	GasUsed     string    `json:"gas_used"`
	GasPrice    string    `json:"gas_price"`
	Network     string    `json:"network"`
}

func main() {
	// Configuration
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	streamName := getEnv("NATS_STREAM_NAME", "TRANSACTIONS")
	subjectPrefix := getEnv("NATS_SUBJECT_PREFIX", "transactions")
	consumerName := getEnv("CONSUMER_NAME", "example-consumer")

	fmt.Printf("Connecting to NATS at %s\n", natsURL)
	fmt.Printf("Stream: %s, Subject: %s.events, Consumer: %s\n", streamName, subjectPrefix, consumerName)

	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.Name("transaction-event-consumer"),
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Check if stream exists
	subject := fmt.Sprintf("%s.events", subjectPrefix)
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil {
		log.Fatalf("Stream %s does not exist: %v", streamName, err)
	}

	fmt.Printf("Stream Info:\n")
	fmt.Printf("  Name: %s\n", streamInfo.Config.Name)
	fmt.Printf("  Subjects: %v\n", streamInfo.Config.Subjects)
	fmt.Printf("  Messages: %d\n", streamInfo.State.Msgs)
	fmt.Printf("  Bytes: %d\n", streamInfo.State.Bytes)
	fmt.Printf("  Last Message Time: %v\n", streamInfo.State.LastTime)
	fmt.Println()

	// Create or get consumer
	consumerConfig := &nats.ConsumerConfig{
		Durable:       consumerName,
		AckPolicy:     nats.AckExplicitPolicy,
		FilterSubject: subject,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		ReplayPolicy:  nats.ReplayInstantPolicy, // Start from now
	}

	consumer, err := js.AddConsumer(streamName, consumerConfig)
	if err != nil {
		// Consumer might already exist, try to get it
		consumer, err = js.ConsumerInfo(streamName, consumerName)
		if err != nil {
			log.Fatalf("Failed to create or get consumer: %v", err)
		}
	}

	fmt.Printf("Consumer Info:\n")
	fmt.Printf("  Name: %s\n", consumer.Name)
	fmt.Printf("  Filter Subject: %s\n", consumer.Config.FilterSubject)
	fmt.Printf("  Pending Messages: %d\n", consumer.NumPending)
	fmt.Printf("  Delivered Messages: %d\n", consumer.Delivered.Consumer)
	fmt.Println()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received, stopping consumer...")
		cancel()
	}()

	// Subscribe to messages
	fmt.Printf("Starting to consume transaction events from subject: %s\n", subject)
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println(strings.Repeat("-", 80))

	messageCount := 0
	subscription, err := js.PullSubscribe(subject, consumerName)
	if err != nil {
		log.Fatalf("Failed to create pull subscription: %v", err)
	}

	// Start consuming messages
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled, stopping consumer")
			return
		default:
			// Fetch messages
			msgs, err := subscription.Fetch(10, nats.MaxWait(1*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue // No messages available, continue polling
				}
				log.Printf("Error fetching messages: %v", err)
				continue
			}

			for _, msg := range msgs {
				messageCount++

				// Parse transaction event
				var txEvent TransactionEvent
				if err := json.Unmarshal(msg.Data, &txEvent); err != nil {
					log.Printf("Failed to unmarshal transaction event: %v", err)
					msg.Nak() // Negative acknowledgment
					continue
				}

				// Process transaction event
				fmt.Printf("[%d] Transaction Event Received:\n", messageCount)
				fmt.Printf("  Hash: %s\n", txEvent.Hash)
				fmt.Printf("  From: %s\n", txEvent.From)
				fmt.Printf("  To: %s\n", txEvent.To)
				fmt.Printf("  Value: %s wei\n", txEvent.Value)
				fmt.Printf("  Block: %s\n", txEvent.BlockNumber)
				fmt.Printf("  Network: %s\n", txEvent.Network)
				fmt.Printf("  Gas Used: %s\n", txEvent.GasUsed)
				fmt.Printf("  Gas Price: %s\n", txEvent.GasPrice)
				fmt.Printf("  Timestamp: %s\n", txEvent.Timestamp.Format(time.RFC3339))
				fmt.Printf("  Subject: %s\n", msg.Subject)
				fmt.Println(strings.Repeat("-", 80))

				// Acknowledge message
				if err := msg.Ack(); err != nil {
					log.Printf("Failed to acknowledge message: %v", err)
				}
			}
		}
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
