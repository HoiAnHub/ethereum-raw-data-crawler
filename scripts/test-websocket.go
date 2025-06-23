package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// Get WebSocket URL from environment or use default
	wsURL := os.Getenv("ETHEREUM_WS_URL")
	if wsURL == "" {
		wsURL = "wss://eth-mainnet.ws.alchemyapi.io/ws/x6DMAG9Zx4vOWVlS3Dov4"
	}

	fmt.Printf("Testing WebSocket connection to: %s\n", wsURL)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal")
		cancel()
	}()

	// Connect to WebSocket
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Successfully connected to WebSocket")

	// Subscribe to new heads
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newHeads"},
	}

	if err := conn.WriteJSON(subscribeMsg); err != nil {
		log.Fatalf("Failed to send subscribe message: %v", err)
	}

	fmt.Println("📡 Sent subscription request for new blocks")

	// Listen for messages
	messageCount := 0
	blockCount := 0
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\n📊 Statistics:\n")
			fmt.Printf("   Duration: %v\n", time.Since(startTime))
			fmt.Printf("   Total messages: %d\n", messageCount)
			fmt.Printf("   Block notifications: %d\n", blockCount)
			if blockCount > 0 {
				avgBlockTime := time.Since(startTime) / time.Duration(blockCount)
				fmt.Printf("   Average block time: %v\n", avgBlockTime)
			}
			return
		default:
			// Set read deadline
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			var message map[string]interface{}
			if err := conn.ReadJSON(&message); err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("❌ WebSocket connection closed: %v\n", err)
					return
				}
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					fmt.Print(".")
					continue
				}
				fmt.Printf("❌ Error reading message: %v\n", err)
				return
			}

			messageCount++
			fmt.Printf("\n📨 Message #%d received at %s\n", messageCount, time.Now().Format("15:04:05"))

			// Pretty print the message
			if jsonBytes, err := json.MarshalIndent(message, "", "  "); err == nil {
				fmt.Printf("   Content: %s\n", string(jsonBytes))
			}

			// Handle subscription confirmation
			if result, ok := message["result"].(string); ok {
				fmt.Printf("🔗 Subscription ID: %s\n", result)
				continue
			}

			// Handle new block notifications
			if method, ok := message["method"].(string); ok && method == "eth_subscription" {
				blockCount++
				fmt.Printf("🆕 Block notification #%d\n", blockCount)

				if params, ok := message["params"].(map[string]interface{}); ok {
					if result, ok := params["result"].(map[string]interface{}); ok {
						if numberHex, ok := result["number"].(string); ok {
							fmt.Printf("   Block number: %s\n", numberHex)
							
							// Convert hex to decimal
							if blockNum, err := parseHexToInt(numberHex); err == nil {
								fmt.Printf("   Block number (decimal): %d\n", blockNum)
							}
						}
						if hash, ok := result["hash"].(string); ok {
							fmt.Printf("   Block hash: %s\n", hash)
						}
						if timestamp, ok := result["timestamp"].(string); ok {
							fmt.Printf("   Timestamp: %s\n", timestamp)
						}
					}
				}
			}
		}
	}
}

func parseHexToInt(hexStr string) (int64, error) {
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	
	var result int64
	for _, char := range hexStr {
		result *= 16
		if char >= '0' && char <= '9' {
			result += int64(char - '0')
		} else if char >= 'a' && char <= 'f' {
			result += int64(char - 'a' + 10)
		} else if char >= 'A' && char <= 'F' {
			result += int64(char - 'A' + 10)
		} else {
			return 0, fmt.Errorf("invalid hex character: %c", char)
		}
	}
	return result, nil
}
