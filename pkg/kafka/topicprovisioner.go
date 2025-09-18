package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type TopicConfig struct {
	Name          string
	Partitions    int
	Replicas      int
	RetentionTime TimeDurationMilliSeconds
}

func (c TopicConfig) Validate() error {
	if c.Name == "" {
		return errors.New("topic name must not be empty")
	}

	if c.Partitions <= 0 {
		return errors.New("number of partitions must be greater than zero")
	}

	return nil
}

type TopicProvisioner interface {
	Provision(ctx context.Context, topics ...TopicConfig) error
	DeProvision(ctx context.Context, topics ...string) error
}

type AdminClient interface {
	CreateTopics(ctx context.Context, topics []kafka.TopicSpecification, options ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error)
	DeleteTopics(ctx context.Context, topics []string, options ...kafka.DeleteTopicsAdminOption) ([]kafka.TopicResult, error)
}

type TopicProvisionerNoop struct{}

var _ TopicProvisioner = (*TopicProvisionerNoop)(nil)

func (t *TopicProvisionerNoop) Provision(ctx context.Context, topics ...TopicConfig) error {
	return nil
}

func (t *TopicProvisionerNoop) DeProvision(ctx context.Context, topics ...string) error {
	return nil
}

type TopicProvisionerConfig struct {
	AdminClient AdminClient
	Logger      *slog.Logger
	Meter       metric.Meter

	// CacheSize stores he maximum number of entries stored in topic cache at a time which after the least recently used is evicted.
	// Setting it to 0 makes the cache size unlimited.
	CacheSize int

	// CacheTTL stores maximum time an entries is kept in cache before being evicted.
	// Setting it to 0 disables cache entry expiration.
	CacheTTL time.Duration

	// ProtectedTopics defines a list of topics which are protected from deletion.
	ProtectedTopics []string
}

// NewTopicProvisioner returns a new TopicProvisioner.
func NewTopicProvisioner(config TopicProvisionerConfig) (TopicProvisioner, error) {
	if config.AdminClient == nil {
		return nil, errors.New("kafka admin client is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if config.Meter == nil {
		return nil, errors.New("meter is required")
	}

	protectedTopics := make(map[string]struct{}, len(config.ProtectedTopics))
	for _, protectedTopic := range config.ProtectedTopics {
		protectedTopics[protectedTopic] = struct{}{}
	}

	provisioner := &topicProvisioner{
		client:          config.AdminClient,
		logger:          config.Logger,
		protectedTopics: protectedTopics,
	}

	provisioner.cache = expirable.NewLRU[string, struct{}](config.CacheSize, provisioner.evictCallback, config.CacheTTL)

	var err error

	provisioner.metrics.Errors, err = config.Meter.Int64Counter(
		"topicprovisioner.errors",
		metric.WithDescription("The number of provision/de-provision operations ended with error"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: topicprovisioner.errors: %w", err)
	}

	provisioner.metrics.Entries, err = config.Meter.Int64Gauge(
		"topicprovisioner.cache_entries",
		metric.WithDescription("The number of entries stored in cache"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: topicprovisioner.cache_entries: %w", err)
	}

	provisioner.metrics.Evictions, err = config.Meter.Int64Counter(
		"topicprovisioner.cache_evictions",
		metric.WithDescription("The number of entries evicted from cache"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: topicprovisioner.cache_evictions: %w", err)
	}

	provisioner.metrics.Lookups, err = config.Meter.Int64Counter(
		"topicprovisioner.cache_lookups",
		metric.WithDescription("The number of cache lookups"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: topicprovisioner.cache_lookups: %w", err)
	}

	provisioner.metrics.MaxSize, err = config.Meter.Int64Gauge(
		"topicprovisioner.cache_max_size",
		metric.WithDescription("The maximum number of entries can be stored in cache"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: topicprovisioner.cache_max_size: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	provisioner.metrics.MaxSize.Record(ctx, int64(config.CacheSize))

	return provisioner, nil
}

var _ TopicProvisioner = (*topicProvisioner)(nil)

// topicProvisioner implements TopicProvisioner interface for creating/deleting Kafka topics. It also utilizes an LRU cache
// to keep track of topics which have been already provisioned. This allows the provisioner being called multiple times with the same
// set of topics without need for extra round-trip to Kafka brokers for sub-sequential calls.
type topicProvisioner struct {
	client AdminClient
	logger *slog.Logger
	cache  *expirable.LRU[string, struct{}]

	protectedTopics map[string]struct{}

	metrics struct {
		// Errors
		Errors metric.Int64Counter
		// Number of entries stored in cache
		Entries metric.Int64Gauge
		// Number of entries evicted from cache
		Evictions metric.Int64Counter
		// Number of cache lookups
		Lookups metric.Int64Counter
		// Maximum number of entries can be stored in cache without eviction
		MaxSize metric.Int64Gauge
	}
}

func (p *topicProvisioner) evictCallback(topic string, _ struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	p.logger.Debug("topic is removed from cache", "topic", topic)

	p.metrics.Evictions.Add(ctx, 1)
}

func (p *topicProvisioner) Provision(ctx context.Context, topics ...TopicConfig) error {
	if len(topics) == 0 {
		return nil
	}

	topicSpecs := make([]kafka.TopicSpecification, 0, len(topics))
	for _, topic := range topics {
		if p.cache.Contains(topic.Name) {
			p.logger.Debug("skip topic as it is already provisioned", "topic", topic.Name)
			continue
		}

		if err := topic.Validate(); err != nil {
			return fmt.Errorf("invalid topic configuration: %w", err)
		}

		topicSpec := kafka.TopicSpecification{
			Topic:         topic.Name,
			NumPartitions: topic.Partitions,
			Config:        make(map[string]string),
		}

		if topic.Replicas > 0 {
			topicSpec.ReplicationFactor = topic.Replicas
		}

		if topic.RetentionTime > 0 {
			topicSpec.Config["retention.ms"] = topic.RetentionTime.String()
		}

		p.logger.Debug("mark topic to be provisioned", "topic", topic.Name)

		topicSpecs = append(topicSpecs, topicSpec)
	}

	p.metrics.Lookups.Add(ctx, int64(len(topics)-len(topicSpecs)), metric.WithAttributes(attribute.String("lookup", "hit")))
	p.metrics.Lookups.Add(ctx, int64(len(topicSpecs)), metric.WithAttributes(attribute.String("lookup", "miss")))

	// Return if all topics are present in cache meaning they are already provisioned.
	if len(topicSpecs) == 0 {
		return nil
	}

	results, err := p.client.CreateTopics(ctx, topicSpecs)
	if err != nil {
		p.metrics.Errors.Add(ctx, 1, metric.WithAttributes(attribute.String("scope", "client")))

		return fmt.Errorf("failed to provision topics: %w", err)
	}

	var errs []error
	for _, result := range results {
		switch result.Error.Code() {
		case kafka.ErrNoError, kafka.ErrTopicAlreadyExists:
			_ = p.cache.Add(result.Topic, struct{}{})

			p.logger.Debug("provisioned topic is added to cache", "topic", result.Topic)
		default:
			p.metrics.Errors.Add(ctx, 1, metric.WithAttributes(attribute.String("scope", "topic")))

			errs = append(errs, fmt.Errorf("failed to create topic %q: %w", result.Topic, result.Error))
		}
	}

	p.metrics.Entries.Record(ctx, int64(p.cache.Len()))

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (p *topicProvisioner) DeProvision(ctx context.Context, topics ...string) error {
	if len(topics) == 0 {
		return nil
	}

	topicsToDelete := make([]string, 0, len(topics))
	for _, topic := range topics {
		if topic == "" {
			p.logger.Warn("skip topic: empty topic name", "topic", topic)

			continue
		}

		// Skip protected topics to avoid accidental deletion
		if _, ok := p.protectedTopics[topic]; ok {
			p.logger.InfoContext(ctx, "skip topic: protected topic", "topic", topic)

			continue
		}

		topicsToDelete = append(topicsToDelete, topic)
	}

	results, err := p.client.DeleteTopics(ctx, topicsToDelete)
	if err != nil {
		p.metrics.Errors.Add(ctx, 1, metric.WithAttributes(attribute.String("scope", "client")))

		return fmt.Errorf("failed to de-provision topics: %w", err)
	}

	var errs []error
	for _, result := range results {
		switch result.Error.Code() {
		case kafka.ErrNoError, kafka.ErrUnknownTopicOrPart:
			_ = p.cache.Remove(result.Topic)

			p.logger.Debug("de-provisioned topic is removed from cache", "topic", result.Topic)
		default:
			p.metrics.Errors.Add(ctx, 1, metric.WithAttributes(attribute.String("scope", "topic")))

			errs = append(errs, fmt.Errorf("failed to delete topic %q: %w", result.Topic, result.Error))
		}
	}

	p.metrics.Entries.Record(ctx, int64(p.cache.Len()))

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
