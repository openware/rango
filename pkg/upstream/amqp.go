package upstream

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type AMQPSession struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

const (
	// When resending messages the server didn't confirm
	resendDelay = 5 * time.Second
)

// NewAMQPSession creates a new consumer state instance, and automatically
// attempts to connect to the server.
func NewAMQPSession(addr string) *AMQPSession {
	session := AMQPSession{}
	err := session.connect(addr)
	if err != nil {
		log.Error().Msgf("Connection to AMQP failed: %s", err.Error())
	}
	return &session
}

// connect will create a new AMQP connection
func (session *AMQPSession) connect(addr string) error {
	conn, err := amqp.Dial(addr)

	if err != nil {
		return err
	}

	session.connection = conn
	session.channel, err = conn.Channel()

	if err != nil {
		return err
	}

	log.Info().Msg("Connected to AMQP!")
	return nil
}

// Push will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// recieve the message.
func (session *AMQPSession) Push(ex, rt string, data []byte) error {
	return session.channel.Publish(
		ex,    // Exchange
		rt,    // Routing key
		false, // Mandatory
		false, // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// Stream will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (session *AMQPSession) Stream(exName, qName string) (<-chan amqp.Delivery, error) {
	_, err := session.channel.QueueDeclare(
		qName,
		false, // Durable
		true,  // Delete when unused
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)

	if err != nil {
		return nil, err
	}

	session.channel.QueueBind(qName, "#", exName, false, nil)

	return session.channel.Consume(
		qName,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// Close will cleanly shutdown the channel and connection.
func (session *AMQPSession) Close() error {
	err := session.channel.Close()
	if err != nil {
		return err
	}
	err = session.connection.Close()
	if err != nil {
		return err
	}
	return nil
}
