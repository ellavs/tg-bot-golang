// Package kafka Хелпер для работы с кафкой
package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/pkg/errors"
	"time"
)

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewSyncProducer(brokerList []string, topic string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	// Waits for all in-sync replicas to commit before responding.
	config.Producer.RequiredAcks = sarama.WaitForAll
	// The total number of times to retry sending a message (default 3).
	config.Producer.Retry.Max = 3
	// How long to wait for the cluster to settle between retries (default 100ms).
	config.Producer.Retry.Backoff = time.Millisecond * 250
	// idempotent producer has a unique producer ID and uses sequence IDs for each message,
	// allowing the broker to ensure, on a per-partition basis, that it is committing ordered messages with no duplication.
	//config.Producer.Idempotent = true
	if config.Producer.Idempotent {
		config.Producer.Retry.Max = 1
		config.Net.MaxOpenRequests = 1
	}
	//  Successfully delivered messages will be returned on the Successes channe
	config.Producer.Return.Successes = true
	// Generates partitioners for choosing the partition to send messages to (defaults to hashing the message key)
	_ = config.Producer.Partitioner

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		return &KafkaProducer{}, errors.Wrap(err, "Starting Sarama producer")
	}

	kafkaProducer := &KafkaProducer{
		producer: producer,
		topic:    topic,
	}

	return kafkaProducer, nil
}

func (k *KafkaProducer) SendMessage(key string, value string) (partition int32, offset int64, err error) {
	msg := sarama.ProducerMessage{
		Topic: k.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(value),
	}
	p, o, err := k.producer.SendMessage(&msg)
	if err != nil {
		return 0, 0, err
	}
	return p, o, nil
}

func (k *KafkaProducer) GetTopic() string {
	return k.topic
}
