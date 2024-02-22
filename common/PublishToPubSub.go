package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
)

// Function to publish message to Pub/Sub
func PublishToPubSub(topicName string, Subscription_Name string, EndPoint string, data map[string]interface{}) error {
	ctx := context.Background()
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Create a Pub/Sub client
	client, err := pubsub.NewClient(ctx, "capstore-takeoff")
	if err != nil {
		return err
	}
	defer client.Close()

	// Get the Pub/Sub topic
	topic := client.Topic(topicName)

	// Check if the topic exists
	exist, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("Failed to check if topic exists: %w", err)
	}
	// If the topic doesn't exist, create it
	if !exist {
		_, err := client.CreateTopic(ctx, topicName)
		if err != nil {
			return fmt.Errorf("Failed to create topic: %w", err)
		}
	}

	// Check if the subscription exists
	sub := client.Subscription(Subscription_Name)
	//sub := client.Subscription("Thumbnail_Subscription")
	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("Failed to check if subscription exists: %w", err)
	}

	// If subscription doesn't exist, create it
	if !exists {
		_, err := client.CreateSubscription(ctx, Subscription_Name, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 10 * time.Second,
			PushConfig: pubsub.PushConfig{
				Endpoint: EndPoint,
				Wrapper: &pubsub.NoWrapper{
					WriteMetadata: false,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("Failed to create subscription: %w", err)
		}
	}

	// Publish the message
	result := topic.Publish(ctx, &pubsub.Message{
		Data: jsonData,
	})

	_, err = result.Get(ctx)
	return err
}
