package amqp

import (
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type AMQPSession struct {
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	streamsReInit   []chan bool
	ready           chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isready         bool
	mutex           sync.Mutex
	mutexCh         sync.Mutex
}

const (
	// When reconnecting to the server after connection failure
	reconnectDelay = 5 * time.Second

	// When setting up the channel after a channel exception
	reInitDelay = 2 * time.Second

	// When resending messages the server didn't confirm
	resendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("session is shutting down")
)

// NewAMQPSession creates a new consumer state instance, and automatically
// attempts to connect to the server.
func NewAMQPSession(addr string) (*AMQPSession, error) {
	session := AMQPSession{
		ready:         make(chan bool, 1),
		streamsReInit: make([]chan bool, 0),
	}
	go session.handleReconnect(addr)
	return &session, nil
}

func (session *AMQPSession) waitChannelReady() {
	session.mutex.Lock()

	if !session.isready {
		session.mutex.Unlock()
		select {
		case <-session.ready:
			return
		}
	} else {
		session.mutex.Unlock()
	}
}

func (session *AMQPSession) setReady(ready bool) {
	session.mutex.Lock()
	session.isready = ready
	session.mutex.Unlock()
	if ready {
		session.ready <- true
	}
}

// connect will create a new AMQP connection
func (session *AMQPSession) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)

	if err != nil {
		return nil, err
	}

	session.changeConnection(conn)
	log.Info().Msg("AMQP Connected to AMQP!")

	return conn, nil
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (session *AMQPSession) handleReconnect(addr string) {
	for {
		session.setReady(false)
		log.Info().Msg("AMQP Attempting to connect")

		conn, err := session.connect(addr)

		if err != nil {
			log.Info().Msg("AMQP Failed to connect. Retrying...")

			select {
			case <-session.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := session.handleReInit(conn); done {
			break
		}
	}
	log.Fatal().Msg("AMQP stopped")
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (session *AMQPSession) handleReInit(conn *amqp.Connection) bool {
	for {
		session.setReady(false)

		err := session.init(conn)

		if err != nil {
			log.Info().Msg("AMQP Failed to initialize channel. Retrying...")

			select {
			case <-session.done:
				log.Info().Msg("AMQP Connection closed. Done")
				return true
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-session.done:
			log.Info().Msg("AMQP Connection closed. Done")
			return true
		case <-session.notifyConnClose:
			log.Info().Msg("AMQP Connection closed. Reconnecting...")
			for _, ch := range session.streamsReInit {
				ch <- true
			}
			return false
		case <-session.notifyChanClose:
			log.Info().Msg("AMQP Channel closed. Re-running init...")
			for _, ch := range session.streamsReInit {
				ch <- true
			}
		}
	}
}

// init will initialize channel
func (session *AMQPSession) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return err
	}

	err = ch.Confirm(false)

	if err != nil {
		return err
	}

	session.changeChannel(ch)
	session.setReady(true)
	log.Info().Msg("AMQP Setup complete!")

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (session *AMQPSession) changeConnection(connection *amqp.Connection) {
	session.connection = connection
	session.notifyConnClose = make(chan *amqp.Error)
	session.connection.NotifyClose(session.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (session *AMQPSession) changeChannel(channel *amqp.Channel) {
	session.mutexCh.Lock()
	session.channel = channel
	session.notifyChanClose = make(chan *amqp.Error)
	session.notifyConfirm = make(chan amqp.Confirmation, 1)
	session.channel.NotifyClose(session.notifyChanClose)
	session.channel.NotifyPublish(session.notifyConfirm)
	session.mutexCh.Unlock()
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously re-sends messages until a confirm is received.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (session *AMQPSession) Push(ex, rt string, data []byte) error {
	for {
		err := session.UnsafePush(ex, rt, data)
		if err != nil {
			log.Info().Msg("AMQP Push failed. Retrying...")
			select {
			case <-session.done:
				return errShutdown
			case <-time.After(resendDelay):
			}
			continue
		}
		select {
		case confirm := <-session.notifyConfirm:
			if confirm.Ack {
				log.Info().Msg("AMQP Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		log.Info().Msg("AMQP Push didn't confirm. Retrying...")
	}
}

// Push will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// recieve the message.
func (session *AMQPSession) UnsafePush(ex, rt string, data []byte) error {
	session.waitChannelReady()
	session.mutexCh.Lock()
	defer session.mutexCh.Unlock()
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
func (session *AMQPSession) Stream(exName, qName string, consumer func(amqp.Delivery)) error {
	session.mutex.Lock()
	reinit := make(chan bool, 1)
	session.streamsReInit = append(session.streamsReInit, reinit)
	session.mutex.Unlock()
	go func() {
		for {
			session.waitChannelReady()

			_, err := session.channel.QueueDeclare(
				qName,
				false, // Durable
				true,  // Delete when unused
				true,  // Exclusive
				false, // No-wait
				nil,   // Arguments
			)

			if err != nil {
				panic("AMQP failed to declare queue " + qName + ": " + err.Error())
			}

			session.channel.ExchangeDeclare(exName, "topic", false, false, false, false, nil)
			session.channel.QueueBind(qName, "#", exName, false, nil)

			ch, err := session.channel.Consume(
				qName,
				"",    // Consumer
				true,  // Auto-Ack
				true,  // Exclusive
				false, // No-local
				false, // No-Wait
				nil,   // Args
			)

			if err != nil {
				panic("AMQP failed to consume queue " + err.Error())
			}

			func() {
				for {
					select {
					case delivery := <-ch:
						consumer(delivery)
					case <-reinit:
						return
					}
				}
			}()

			log.Warn().Msg("AMQP Stream loop interrupted")
		}
	}()
	return nil
}

// Close will delete the queue, close the channel and the connection.
func (session *AMQPSession) Close(qName string) error {
	log.Error().Msg("Closing connection to RabbitMQ")
	_, err := session.channel.QueueDelete(qName, false, false, false)
	if err != nil {
		return err
	}
	err = session.channel.Close()
	if err != nil {
		return err
	}
	err = session.connection.Close()
	if err != nil {
		return err
	}
	return nil
}
