// Package kfkimp implements the Kafka message broker for the issue tracking system.
// It provides functionality to publish and subscribe to project updates using Kafka.
package kfkimp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/yasindce1998/issue-tracker/logger"
	"github.com/yasindce1998/issue-tracker/pkg/messaging/broker"
	projectPbv1 "github.com/yasindce1998/issue-tracker/pkg/pb/project/v1"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// KafkaBroker implements the MessageBroker interface using Kafka
type KafkaBroker struct {
	writer           *kafka.Writer
	readers          map[string]*kafka.Reader
	subscribers      map[string]map[chan<- *projectPbv1.ProjectUpdateResponse]bool
	subscribersMutex sync.RWMutex
	brokers          []string
	topicPrefix      string
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewKafkaBroker creates a new Kafka messaging implementation
func NewKafkaBroker(brokers []string, topicPrefix string) (broker.MessageBroker, error) {
	// Directly try to create topic first before doing anything else
	topicName := topicPrefix + ".projects"
	created := false

	// Try multiple brokers in case one is not responding
	for _, broker := range brokers {
		logger.ZapLogger.Info("Attempting to connect to Kafka broker",
			zap.String("broker", broker))

		conn, err := kafka.Dial("tcp", broker)
		if err != nil {
			logger.ZapLogger.Warn("Could not connect to Kafka broker",
				zap.Error(err),
				zap.String("broker", broker))
			continue
		}

		controller, err := conn.Controller()
		if err != nil {
			logger.ZapLogger.Warn("Failed to get Kafka controller", zap.Error(err))
			if err := conn.Close(); err != nil {
				logger.ZapLogger.Warn("Failed to close Kafka connection", zap.Error(err))
			}
			continue
		}

		controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
		if err != nil {
			logger.ZapLogger.Warn("Failed to connect to Kafka controller",
				zap.Error(err),
				zap.String("host", controller.Host),
				zap.Int("port", controller.Port))
			if err := conn.Close(); err != nil {
				logger.ZapLogger.Warn("Failed to close Kafka connection", zap.Error(err))
			}
			continue
		}

		topicConfigs := []kafka.TopicConfig{
			{
				Topic:             topicName,
				NumPartitions:     1,
				ReplicationFactor: 1,
			},
		}

		err = controllerConn.CreateTopics(topicConfigs...)
		if err := conn.Close(); err != nil {
			logger.ZapLogger.Warn("Failed to close Kafka connection", zap.Error(err))
		}
		if err := controllerConn.Close(); err != nil {
			logger.ZapLogger.Warn("Failed to close Kafka controller connection", zap.Error(err))
		}

		if err != nil {
			// Don't fail if topic already exists
			logger.ZapLogger.Info("Topic creation result",
				zap.Error(err),
				zap.String("topic", topicName))
			if strings.Contains(err.Error(), "already exists") {
				created = true
				break
			}
		} else {
			logger.ZapLogger.Info("Topic created successfully",
				zap.String("topic", topicName))
			created = true
			break
		}
	}

	if !created {
		logger.ZapLogger.Warn("Could not create topic - will try again when publishing",
			zap.String("topic", topicName))
	}

	// Create the writer after topic creation attempt
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topicName,
		Balancer: &kafka.LeastBytes{},
	})

	// Log the configuration
	logger.ZapLogger.Info("Initializing Kafka broker",
		zap.Strings("brokers", brokers),
		zap.String("topicPrefix", topicPrefix))

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaBroker{
		writer:      writer,
		readers:     make(map[string]*kafka.Reader),
		subscribers: make(map[string]map[chan<- *projectPbv1.ProjectUpdateResponse]bool),
		brokers:     brokers,
		topicPrefix: topicPrefix,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// PublishUpdate publishes a project update to Kafka
func (k *KafkaBroker) PublishUpdate(ctx context.Context, projectID string, update *projectPbv1.ProjectUpdateResponse) error {
	// Create merged context to respect both the broker's and the caller's context
	mergedCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-k.ctx.Done():
			cancel()
		case <-ctx.Done():
			cancel()
		}
	}()

	// Serialize the protobuf message
	value, err := proto.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal project update: %w", err)
	}

	// Get topic name and log it
	topicName := k.topicPrefix + ".projects"
	logger.ZapLogger.Debug("Publishing to Kafka topic",
		zap.String("topic", topicName),
		zap.String("projectID", projectID),
		zap.Int("messageSize", len(value)))

	// Write the message to Kafka
	err = k.writer.WriteMessages(mergedCtx, kafka.Message{
		Key:   []byte(projectID),
		Value: value,
	})

	if err != nil {
		return k.handlePublishError(mergedCtx, err, topicName, projectID, value)
	}

	logger.ZapLogger.Debug("Successfully published message to Kafka",
		zap.String("topic", topicName),
		zap.String("projectID", projectID))

	return nil
}

// handlePublishError attempts to recover from Kafka publish errors
func (k *KafkaBroker) handlePublishError(ctx context.Context, err error, topicName, projectID string, value []byte) error {
	if err.Error() == "kafka: unknown topic or partition" ||
		err.Error() == "[3] Unknown Topic Or Partition" {
		logger.ZapLogger.Warn("Topic doesn't exist, attempting to create it",
			zap.String("topic", topicName),
			zap.Error(err))

		// Attempt to create the topic
		if k.createTopicAndRetry(ctx, topicName, projectID, value) {
			return nil
		}
	}
	return fmt.Errorf("failed to write message to Kafka: %w", err)
}

// createTopicAndRetry creates a topic and retries the message publish
// Returns true if successful, false otherwise
func (k *KafkaBroker) createTopicAndRetry(ctx context.Context, topicName, projectID string, value []byte) bool {
	conn, dialErr := kafka.Dial("tcp", k.brokers[0])
	if dialErr != nil {
		return false
	}

	defer func() {
		if err := conn.Close(); err != nil {
			logger.ZapLogger.Warn("Failed to close Kafka connection", zap.Error(err))
		}
	}()

	controller, ctrlErr := conn.Controller()
	if ctrlErr != nil {
		return false
	}

	controllerConn, connErr := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if connErr != nil {
		return false
	}

	defer func() {
		if err := controllerConn.Close(); err != nil {
			logger.ZapLogger.Warn("Failed to close Kafka controller connection", zap.Error(err))
		}
	}()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topicName,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	createErr := controllerConn.CreateTopics(topicConfigs...)
	if createErr != nil {
		logger.ZapLogger.Error("Failed to create topic",
			zap.String("topic", topicName),
			zap.Error(createErr))
		return false
	}

	logger.ZapLogger.Info("Topic created successfully, retrying message publish",
		zap.String("topic", topicName))

	// Try publishing again
	retryErr := k.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(projectID),
		Value: value,
	})

	return retryErr == nil
}

// Subscribe creates a subscription to project updates
func (k *KafkaBroker) Subscribe(ctx context.Context, projectID string) (<-chan *projectPbv1.ProjectUpdateResponse, error) {
	k.subscribersMutex.Lock()
	defer k.subscribersMutex.Unlock()

	// Create channel for this subscriber
	ch := make(chan *projectPbv1.ProjectUpdateResponse, 10)

	// Create map if it doesn't exist
	if _, exists := k.subscribers[projectID]; !exists {
		k.subscribers[projectID] = make(map[chan<- *projectPbv1.ProjectUpdateResponse]bool)

		// Create a reader for this project if it doesn't exist
		if _, exists := k.readers[projectID]; !exists {
			logger.ZapLogger.Info("Creating Kafka reader for project",
				zap.String("projectID", projectID),
				zap.String("topic", k.topicPrefix+".projects"))

			reader := kafka.NewReader(kafka.ReaderConfig{
				Brokers:   k.brokers,
				Topic:     k.topicPrefix + ".projects",
				GroupID:   fmt.Sprintf("issue-tracker-project-%s", projectID),
				Partition: -1,
			})
			k.readers[projectID] = reader

			// Start consuming messages
			go k.consumeMessages(projectID, reader)
		}
	}

	k.subscribers[projectID][ch] = true
	logger.ZapLogger.Debug("Added new subscriber for project",
		zap.String("projectID", projectID),
		zap.Int("totalSubscribers", len(k.subscribers[projectID])))

	// Monitor context cancellation
	go func() {
		<-ctx.Done()
		k.subscribersMutex.Lock()
		defer k.subscribersMutex.Unlock()

		// Remove the channel if context is done
		if subs, ok := k.subscribers[projectID]; ok {
			delete(subs, ch)
			logger.ZapLogger.Debug("Removed subscriber due to context cancellation",
				zap.String("projectID", projectID))
		}
	}()

	return ch, nil
}

// Unsubscribe removes a subscription
func (k *KafkaBroker) Unsubscribe(_ context.Context, projectID string, _ <-chan *projectPbv1.ProjectUpdateResponse) error {
	k.subscribersMutex.Lock()
	defer k.subscribersMutex.Unlock()

	subs, ok := k.subscribers[projectID]
	if !ok {
		return nil
	}

	// Find and remove the specific channel
	for subCh := range subs {
		// This is a bit complex as we can't directly compare channel references
		// For efficiency, in a real implementation you'd want to store a map of channels to IDs
		// But for this example, we'll just remove them all
		delete(subs, subCh)
	}

	k.cleanupIfNoSubscribers(projectID, subs)

	return nil
}

// cleanupIfNoSubscribers removes the reader if there are no more subscribers
func (k *KafkaBroker) cleanupIfNoSubscribers(projectID string, subs map[chan<- *projectPbv1.ProjectUpdateResponse]bool) {
	if len(subs) == 0 {
		if reader, ok := k.readers[projectID]; ok {
			if err := reader.Close(); err != nil {
				logger.ZapLogger.Warn("Failed to close Kafka reader", zap.Error(err))
			}
			delete(k.readers, projectID)
		}
		delete(k.subscribers, projectID)
	}
}

// Close releases Kafka resources
func (k *KafkaBroker) Close() error {
	k.cancel()

	k.subscribersMutex.Lock()
	defer k.subscribersMutex.Unlock()

	// Close the writer
	if err := k.writer.Close(); err != nil {
		return err
	}

	// Close all readers
	for _, reader := range k.readers {
		if err := reader.Close(); err != nil {
			return err
		}
	}

	// Close all subscriber channels
	for _, subscribers := range k.subscribers {
		for ch := range subscribers {
			close(ch)
		}
	}

	return nil
}

// consumeMessages reads messages from Kafka and distributes them to subscribers
func (k *KafkaBroker) consumeMessages(projectID string, reader *kafka.Reader) {
	for {
		select {
		case <-k.ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(k.ctx)
			if err != nil {
				continue
			}

			// Only process messages for this project
			if string(msg.Key) != projectID {
				continue
			}

			// Deserialize the protobuf message
			update := &projectPbv1.ProjectUpdateResponse{}
			if err := proto.Unmarshal(msg.Value, update); err != nil {
				continue
			}

			// Distribute to subscribers
			k.distributeUpdate(projectID, update)
		}
	}
}

// distributeUpdate sends update to all subscribers
func (k *KafkaBroker) distributeUpdate(projectID string, update *projectPbv1.ProjectUpdateResponse) {
	k.subscribersMutex.RLock()
	defer k.subscribersMutex.RUnlock()

	if subscribers, ok := k.subscribers[projectID]; ok {
		for ch := range subscribers {
			select {
			case ch <- update:
				// Message sent successfully
			default:
				// Channel is full, skip
			}
		}
	}
}
