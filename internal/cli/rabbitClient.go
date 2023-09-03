package cli

import (
	"context"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
	Uri string
}

var connection *amqp.Connection
var channel *amqp.Channel
var lock = &sync.Mutex{}

// Get client form configs. Go routines safe Singleton implementation
func getRabbitClient(rc *RabbitClient) (*amqp.Connection, *amqp.Channel, error) {
	if connection == nil {
		lock.Lock()
		defer lock.Unlock()

		// Condition to safely make sure to lock once
		if connection == nil {
			connection, err := amqp.Dial(rc.Uri)
			if err != nil {
				return nil, nil, err
			}

			channel, err := connection.Channel()
			if err != nil {
				return nil, nil, err
			}

			// Queues Declarations
			_, err = channel.QueueDeclare(
				"sound_upload", // queue name
				false,          // durable
				false,          // delete when unused
				false,          // exclusive
				false,          // no-wait
				nil,            // arguments
			)
			if err != nil {
				return nil, nil, err
			}

			_, err = channel.QueueDeclare(
				"transcription_upload",
				false,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				return nil, nil, err
			}

			return connection, channel, nil
		}
		return connection, channel, nil
	}
	return connection, channel, nil
}

func (rc *RabbitClient) EmitUploadEvent(ctx context.Context, msg string) error {
	_, ch, err := getRabbitClient(rc)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		ctx,
		"",
		"sound_upload",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	if err != nil {
		return err
	}

	return nil
}

func (rc *RabbitClient) EmitTrasciptionEvent(ctx context.Context, msg string) error {
	_, ch, err := getRabbitClient(rc)
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(
		ctx,
		"",
		"transcription_upload",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	if err != nil {
		return err
	}

	return nil
}

func (rc *RabbitClient) GetSoundUploadChannel() (<-chan amqp.Delivery, error) {
	_, ch, err := getRabbitClient(rc)
	if err != nil {
		return nil, err
	}

	messagesRabbit, err := ch.Consume(
		"sound_upload",
		"Vosk-Consumer", // Consumer
		false,           // Auto-ack
		false,           // Exclusive
		false,           // No-local
		false,           // No-wait
		nil,             // Args
	)
	if err != nil {
		return nil, err
	}

	return messagesRabbit, nil
}

func (rc *RabbitClient) GetTranscriptionUploadChannel() (<-chan amqp.Delivery, error) {
	_, ch, err := getRabbitClient(rc)
	if err != nil {
		return nil, err
	}

	messagesRabbit, err := ch.Consume(
		"transcription_upload",
		"Gpt-Consumer", // Consumer
		false,          // Auto-ack
		false,          // Exclusive
		false,          // No-local
		false,          // No-wait
		nil,            // Args
	)
	if err != nil {
		return nil, err
	}

	return messagesRabbit, nil
}
