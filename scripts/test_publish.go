// +build ignore

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type LocationEnrichEvent struct {
	PropertyID   uuid.UUID `json:"property_id"`
	Country      string    `json:"country"`
	Region       *string   `json:"region,omitempty"`
	Province     *string   `json:"province,omitempty"`
	City         *string   `json:"city,omitempty"`
	District     *string   `json:"district,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	Street       *string   `json:"street,omitempty"`
	HouseNumber  *string   `json:"house_number,omitempty"`
	PostalCode   *string   `json:"postal_code,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
}

func ptr[T any](v T) *T {
	return &v
}

func main() {
	redisAddr := flag.String("redis", "localhost:6380", "Redis address for streams")
	flag.Parse()

	client := redis.NewClient(&redis.Options{
		Addr: *redisAddr,
	})
	defer client.Close()

	ctx := context.Background()

	// Проверка подключения
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Тестовое событие (Barcelona address)
	event := LocationEnrichEvent{
		PropertyID:   uuid.New(),
		Country:      "España",
		Province:     ptr("Barcelona"),
		City:         ptr("Barcelona"),
		District:     ptr("Gràcia"),
		Neighborhood: ptr("Vila de Gràcia"),
		Street:       ptr("Calle del Torrent de l'Olla"),
		Latitude:     ptr(41.4027042),
		Longitude:    ptr(2.1599563),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Fatalf("Failed to marshal event: %v", err)
	}

	// Публикация в стрим
	result, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: "stream:location:enrich",
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()
	if err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}

	fmt.Printf("✅ Event published successfully!\n")
	fmt.Printf("   Stream: stream:location:enrich\n")
	fmt.Printf("   Message ID: %s\n", result)
	fmt.Printf("   Property ID: %s\n", event.PropertyID)
	fmt.Printf("   Location: %s, %s\n", *event.City, event.Country)
	fmt.Printf("   Coordinates: %.6f, %.6f\n", *event.Latitude, *event.Longitude)

	// Ожидание ответа
	fmt.Printf("\n⏳ Waiting for response in stream:location:done...\n")

	// Создаем consumer group если не существует
	client.XGroupCreateMkStream(ctx, "stream:location:done", "test-consumer", "$")

	// Читаем ответ
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Println("❌ Timeout waiting for response")
			return
		case <-ticker.C:
			results, err := client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{"stream:location:done", "0"},
				Count:   10,
				Block:   0,
			}).Result()
			
			if err != nil && err != redis.Nil {
				continue
			}

			for _, stream := range results {
				for _, msg := range stream.Messages {
					dataStr, ok := msg.Values["data"].(string)
					if !ok {
						continue
					}

					var response map[string]interface{}
					if err := json.Unmarshal([]byte(dataStr), &response); err != nil {
						continue
					}

					if propID, ok := response["property_id"].(string); ok {
						if propID == event.PropertyID.String() {
							fmt.Printf("\n✅ Response received!\n")
							prettyJSON, _ := json.MarshalIndent(response, "", "  ")
							fmt.Printf("%s\n", prettyJSON)
							return
						}
					}
				}
			}
		}
	}
}
