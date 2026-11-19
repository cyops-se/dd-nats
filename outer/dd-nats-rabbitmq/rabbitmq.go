package main

import (
	"encoding/json"
	"fmt"
	"time"

	"dd-nats/common/types"

	"github.com/sirius1024/go-amqp-reconnect/rabbitmq"
	"github.com/streadway/amqp"
)

type RabbitMQEmitter struct {
	Urls        []string `json:"urls"`
	ChannelName string   `json:"channel"`
	Durable     bool     `json:"durable"`

	connection *rabbitmq.Connection
	channel    *rabbitmq.Channel
	queue      amqp.Queue
	messages   chan types.DataPoint
}

type RabbitMQDataPoint struct {
	Signal    string    `json:"signal"`
	Value     float64   `json:"value"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type RabbitMQMetaItem struct {
	Signal      string  `json:"signal"`
	Description string  `json:"description"`
	Dimension   string  `json:"dimension"`
	Min         float64 `json:"range_min"`
	Max         float64 `json:"range_max"`
	Deadband    float64 `json:"deadband"`
}

// InitEmitter starts the emitter, the connection is established in the
// background and retried until it succeeds
func (emitter *RabbitMQEmitter) InitEmitter() error {
	if len(emitter.Urls) == 0 {
		return svc.Error("RabbitMQ init", "Failed to connect RabbitMQ cluster, urls parameter empty")
	}

	emitter.messages = make(chan types.DataPoint, 2000)
	go emitter.processMessages()
	return nil
}

func (emitter *RabbitMQEmitter) connect() error {
	connection, err := rabbitmq.DialCluster(emitter.Urls)
	if err != nil {
		return fmt.Errorf("failed to connect RabbitMQ cluster: %v", err)
	}

	channel, err := connection.Channel()
	if err != nil {
		connection.Close()
		return fmt.Errorf("failed to open channel: %v", err)
	}

	queue, err := channel.QueueDeclare(
		emitter.ChannelName, // name
		emitter.Durable,     // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)

	if err != nil {
		channel.Close()
		connection.Close()
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	emitter.connection = connection
	emitter.channel = channel
	emitter.queue = queue
	svc.Info("RabbitMQ emitter", "RabbitMQ server connected, queue: %s", queue.Name)
	return nil
}

func (emitter *RabbitMQEmitter) processDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err != nil {
		return fmt.Errorf("failed to unmarshal data point: %s", err.Error())
	}

	emitter.ProcessMessage(dp)
	return nil
}

func (emitter *RabbitMQEmitter) ProcessMessage(dp types.DataPoint) {
	if emitter.messages != nil {
		emitter.messages <- dp
	}
}

func (emitter *RabbitMQEmitter) processMessages() {
	// The connection library reconnects automatically once connected, but the
	// initial connection has to be retried here
	for emitter.channel == nil {
		if err := emitter.connect(); err != nil {
			svc.Error("RabbitMQ init", "%s, retrying in 10 seconds", err.Error())
			time.Sleep(10 * time.Second)
		}
	}

	for dp := range emitter.messages {
		// Use safe marshalling to avoid human mistakes when formatting JSON
		rmdp := &RabbitMQDataPoint{Signal: dp.Name, Value: dp.Value, Status: dp.Quality, Timestamp: dp.Time}
		body, err := json.Marshal(rmdp)
		if err != nil {
			svc.Error("RabbitMQ emitter", "Failed to marshal data point %s: %s", dp.Name, err.Error())
			continue
		}

		err = emitter.channel.Publish(
			"",                 // exchange
			emitter.queue.Name, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})

		if err != nil {
			svc.Error("RabbitMQ emitter", "Failed to publish message: %v", err.Error())
		}
	}
}

func (emitter *RabbitMQEmitter) sendMetaRabbit(dp *types.DataPointMeta) {
	msg := &RabbitMQMetaItem{
		Signal:      dp.Name,
		Description: dp.Description,
		Dimension:   dp.EngUnit,
		Max:         dp.MaxValue,
		Min:         dp.MinValue,
	}

	body, _ := json.Marshal(msg)

	err := emitter.channel.Publish(
		"",            // exchange
		"me-metadata", // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		svc.Error("RabbitMQ emitter", "Failed to publish meta message: %v", err.Error())
	}
}
