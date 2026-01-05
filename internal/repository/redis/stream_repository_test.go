package redis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	redisRepo "github.com/location-microservice/internal/repository/redis"
)

// getTestRedisClient creates a Redis client for testing
func getTestRedisClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test connection
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skipf("Redis not available for integration tests: %v", err)
	}

	// Clean up any existing test streams
	client.Del(ctx, "test:stream:location:enrich", "test:stream:location:done")

	return client
}

// TestStreamRepository_CreateConsumerGroup tests consumer group creation
func TestStreamRepository_CreateConsumerGroup(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	logger := zap.NewNop()
	repo := redisRepo.NewStreamRepository(client, logger)
	ctx := context.Background()

	streamName := "test:stream:location:enrich"
	groupName := "test-group"

	// Clean up
	defer func() {
		client.Del(ctx, streamName)
	}()

	// Create consumer group
	err := repo.CreateConsumerGroup(ctx, streamName, groupName)
	require.NoError(t, err)

	// Verify group was created
	groups, err := client.XInfoGroups(ctx, streamName).Result()
	require.NoError(t, err)
	assert.Len(t, groups, 1)
	assert.Equal(t, groupName, groups[0].Name)

	// Creating again should not error (BUSYGROUP handled)
	err = repo.CreateConsumerGroup(ctx, streamName, groupName)
	assert.NoError(t, err)
}

// TestStreamRepository_PublishToStream tests message publishing
func TestStreamRepository_PublishToStream(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	logger := zap.NewNop()
	repo := redisRepo.NewStreamRepository(client, logger)
	ctx := context.Background()

	streamName := "test:stream:location:done"

	// Clean up
	defer func() {
		client.Del(ctx, streamName)
	}()

	// Create test event
	propertyID := uuid.New()
	event := &domain.LocationDoneEvent{
		PropertyID: propertyID,
		EnrichedLocation: &domain.EnrichedLocation{
			CityID: ptrInt64(100),
		},
		NearestTransport: []domain.NearestStation{
			{
				StationID: 500,
				Name:      "Test Station",
				Type:      "metro",
				Distance:  350.5,
				LineIDs:   []int64{3, 4},
			},
		},
	}

	// Publish to stream
	err := repo.PublishToStream(ctx, streamName, event)
	require.NoError(t, err)

	// Verify message was published
	messages, err := client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{streamName, "0"},
		Count:   1,
	}).Result()
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Len(t, messages[0].Messages, 1)

	// Verify message content
	msg := messages[0].Messages[0]
	dataStr, ok := msg.Values["data"].(string)
	require.True(t, ok)

	var receivedEvent domain.LocationDoneEvent
	err = json.Unmarshal([]byte(dataStr), &receivedEvent)
	require.NoError(t, err)
	assert.Equal(t, propertyID, receivedEvent.PropertyID)
	assert.NotNil(t, receivedEvent.EnrichedLocation)
	assert.Equal(t, int64(100), *receivedEvent.EnrichedLocation.CityID)
	assert.Len(t, receivedEvent.NearestTransport, 1)
	assert.Equal(t, "Test Station", receivedEvent.NearestTransport[0].Name)
}

// TestStreamRepository_ConsumeStream tests message consumption
func TestStreamRepository_ConsumeStream(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	logger := zap.NewNop()
	repo := redisRepo.NewStreamRepository(client, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	streamName := "test:stream:location:enrich"
	groupName := "test-consumer-group"
	consumerName := "test-consumer"

	// Clean up
	defer func() {
		client.Del(context.Background(), streamName)
	}()

	// Create consumer group
	err := repo.CreateConsumerGroup(ctx, streamName, groupName)
	require.NoError(t, err)

	// Publish a test message
	propertyID := uuid.New()
	testEvent := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		City:       ptrString("Barcelona"),
		Latitude:   ptrFloat64(41.3851),
		Longitude:  ptrFloat64(2.1734),
	}

	err = repo.PublishToStream(ctx, streamName, testEvent)
	require.NoError(t, err)

	// Consume messages
	msgChan, err := repo.ConsumeStream(ctx, streamName, groupName, consumerName)
	require.NoError(t, err)

	// Read message from channel
	select {
	case msg := <-msgChan:
		assert.NotEmpty(t, msg.ID)
		
		// Verify message content
		var receivedEvent domain.LocationEnrichEvent
		err = json.Unmarshal([]byte(msg.Data), &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, propertyID, receivedEvent.PropertyID)
		assert.Equal(t, "Spain", receivedEvent.Country)
		assert.Equal(t, "Barcelona", *receivedEvent.City)
		
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestStreamRepository_AckMessage tests message acknowledgment
func TestStreamRepository_AckMessage(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	logger := zap.NewNop()
	repo := redisRepo.NewStreamRepository(client, logger)
	ctx := context.Background()

	streamName := "test:stream:location:enrich"
	groupName := "test-ack-group"
	consumerName := "test-consumer"

	// Clean up
	defer func() {
		client.Del(ctx, streamName)
	}()

	// Create consumer group
	err := repo.CreateConsumerGroup(ctx, streamName, groupName)
	require.NoError(t, err)

	// Publish a test message
	testEvent := &domain.LocationEnrichEvent{
		PropertyID: uuid.New(),
		Country:    "Spain",
	}
	err = repo.PublishToStream(ctx, streamName, testEvent)
	require.NoError(t, err)

	// Read message
	messages, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: consumerName,
		Streams:  []string{streamName, ">"},
		Count:    1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Len(t, messages[0].Messages, 1)

	messageID := messages[0].Messages[0].ID

	// Check pending messages before ACK
	pending, err := client.XPending(ctx, streamName, groupName).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), pending.Count)

	// Acknowledge message
	err = repo.AckMessage(ctx, streamName, groupName, messageID)
	require.NoError(t, err)

	// Check pending messages after ACK
	pending, err = client.XPending(ctx, streamName, groupName).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending.Count)
}

// TestStreamRepository_ConsumeStream_ContextCancellation tests graceful shutdown
func TestStreamRepository_ConsumeStream_ContextCancellation(t *testing.T) {
	client := getTestRedisClient(t)
	defer client.Close()

	logger := zap.NewNop()
	repo := redisRepo.NewStreamRepository(client, logger)
	ctx, cancel := context.WithCancel(context.Background())

	streamName := "test:stream:location:enrich"
	groupName := "test-cancel-group"
	consumerName := "test-consumer"

	// Clean up
	defer func() {
		client.Del(context.Background(), streamName)
	}()

	// Create consumer group
	err := repo.CreateConsumerGroup(ctx, streamName, groupName)
	require.NoError(t, err)

	// Start consuming
	msgChan, err := repo.ConsumeStream(ctx, streamName, groupName, consumerName)
	require.NoError(t, err)

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Channel should close when context is cancelled
	timeout := time.After(2 * time.Second)
	select {
	case _, ok := <-msgChan:
		if ok {
			// Received a message (ok if we get lucky with timing)
			// Continue to wait for channel close
			select {
			case _, ok := <-msgChan:
				assert.False(t, ok, "Channel should be closed")
			case <-timeout:
				t.Fatal("Channel not closed after context cancellation")
			}
		} else {
			// Channel closed as expected
			assert.False(t, ok)
		}
	case <-timeout:
		t.Fatal("Timeout waiting for channel to close")
	}
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt64(i int64) *int64 {
	return &i
}
