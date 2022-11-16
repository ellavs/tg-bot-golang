// Package kafka Хелпер для работы с кафкой
package kafka

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"log"
)

type KafkaConsumer struct {
	ctx      context.Context
	consumer sarama.ConsumerGroup
	topic    string
}

func NewConsumer(ctx context.Context, brokerList []string, topic string) (*KafkaConsumer, error) {

	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}

	// Create consumer group
	kafkaConsumerGroup := topic + "-consumer-group"
	consumerGroup, err := sarama.NewConsumerGroup(brokerList, kafkaConsumerGroup, config)
	if err != nil {
		return &KafkaConsumer{}, errors.Wrap(err, "Starting consumer group")
	}

	kafkaConsumer := &KafkaConsumer{
		ctx:      ctx,
		consumer: consumerGroup,
		topic:    topic,
	}

	return kafkaConsumer, nil
}

func (c *KafkaConsumer) RunConsume(handlerFunc func(ctx context.Context, key string, value string) error) error {
	consumerGroupHandler := Consumer{
		ctx:         c.ctx,
		handlerFunc: handlerFunc,
	}
	err := c.consumer.Consume(c.ctx, []string{c.topic}, &consumerGroupHandler)
	if err != nil {
		return errors.Wrap(err, "consuming via handler")
	}
	return nil
}

// Consumer represents a Sarama consumer group consumer.
type Consumer struct {
	ctx         context.Context
	handlerFunc func(ctx context.Context, key string, value string) error
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	log.Println("consumer - setup")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Println("consumer - cleanup")
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		err := consumer.handlerFunc(consumer.ctx, string(message.Key), string(message.Value))
		if err == nil {
			session.MarkMessage(message, "")
		}
	}
	return nil
}
