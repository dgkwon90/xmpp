package rabbitmq

import (
	"context"
	"errors"
	"time"

	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// ReconnectDelay When reconnecting to the server after connection failure
	ReconnectDelay = 5 * time.Second
	// ReInitDelay When setting up the channel after a channel exception
	ReInitDelay = 2 * time.Second
	// ResendDelay When resending messages the server didn't confirm
	ResendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("client is shutting down")
)

type Client struct {
	name            string
	queueName       string
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         bool
}

// New creates a new consumer state instance, and automatically
// attempts to connect to the server.
func NewClient(name, queueName, addr string) *Client {
	client := Client{
		//logger:    log.New(os.Stdout, "", log.LstdFlags),
		name:      name,
		queueName: queueName,
		done:      make(chan bool),
	}
	go client.handleReconnect(addr)
	return &client
}

func (client *Client) IsReady() bool {
	return client.isReady
}

func (client *Client) ExchangeDeclare(exchange, exchangeType string) error {
	exchangeDeclareErr := client.channel.ExchangeDeclare(
		exchange,     // name
		exchangeType, // type
		false,        // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if exchangeDeclareErr != nil {
		return exchangeDeclareErr
	}
	return nil
}

func (client *Client) QueueBind(exchange, routingKey string) error {
	queueBindErr := client.channel.QueueBind(
		client.queueName, // queue name
		routingKey,       // routing key
		exchange,         // exchange
		false,            // no-waite
		nil,              // arguments
	)
	if queueBindErr != nil {
		return queueBindErr
	}
	return nil
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (client *Client) handleReconnect(addr string) {
	for {
		client.isReady = false
		log.Printf("Attempting to connect: %v", addr)
		conn, err := client.connect(addr)
		if err != nil {
			log.Printf("Failed to connect. Retrying...: %v", err)
			select {
			case <-client.done:
				return
			case <-time.After(ReconnectDelay):
			}
			continue
		}

		if done := client.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (client *Client) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)

	if err != nil {
		return nil, err
	}

	client.changeConnection(conn)
	log.Println("Connected!")
	return conn, nil
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (client *Client) handleReInit(conn *amqp.Connection) bool {
	for {
		client.isReady = false

		err := client.init(conn)

		if err != nil {
			log.Println("Failed to initialize channel. Retrying...")

			select {
			case <-client.done:
				return true
			case <-time.After(ReInitDelay):
			}
			continue
		}

		select {
		case <-client.done:
			return true
		case <-client.notifyConnClose:
			log.Println("Connection closed. Reconnecting...")
			return false
		case <-client.notifyChanClose:
			log.Println("Channel closed. Re-running init...")
		}
	}
}

// init will initialize channel & declare queue
func (client *Client) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	err = ch.Confirm(false)
	if err != nil {
		log.Println(err)
		return err
	}

	_, err = ch.QueueDeclare(
		client.queueName, // Name
		false,            // Durable
		false,            // Delete when unused
		false,            // Exclusive
		false,            // No-wait
		nil,              // Arguments
	)
	if err != nil {
		log.Println(err)
		return err
	}

	client.changeChannel(ch)
	client.isReady = true
	log.Println("Setup!")

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (client *Client) changeConnection(connection *amqp.Connection) {
	client.connection = connection
	client.notifyConnClose = make(chan *amqp.Error)
	client.connection.NotifyClose(client.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (client *Client) changeChannel(channel *amqp.Channel) {
	client.channel = channel
	client.notifyChanClose = make(chan *amqp.Error)
	client.notifyConfirm = make(chan amqp.Confirmation, 1)
	client.channel.NotifyClose(client.notifyChanClose)
	client.channel.NotifyPublish(client.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously re-sends messages until a confirm is received.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (client *Client) Push(exchangeName, routingKey string, mandatory, immediate bool, pubMsg amqp.Publishing) error {
	if !client.isReady {
		return errors.New("failed to push: not connected")
	}
	for {
		err := client.UnsafePush(exchangeName, routingKey, mandatory, immediate, pubMsg)
		if err != nil {
			log.Printf("Push failed. Retrying...: %v", err)
			select {
			case <-client.done:
				return errShutdown
			case <-time.After(ResendDelay):
			}
			continue
		}
		select {
		case confirm := <-client.notifyConfirm:
			if confirm.Ack {
				log.Println("Push confirmed!")
				return nil
			}
		case <-time.After(ResendDelay):
		}
		log.Println("Push didn't confirm. Retrying...")
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// receive the message.
func (client *Client) UnsafePush(exchangeName string, routingKey string, mandatory, immediate bool, pubMsg amqp.Publishing) error {
	if !client.isReady {
		return errNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// if routingKey is nil, set client queueName
	if len(routingKey) <= 0 {
		routingKey = client.queueName
	}

	return client.channel.PublishWithContext(
		ctx,          //context
		exchangeName, // Exchange
		routingKey,   // Routing key
		mandatory,    // Mandatory
		immediate,    // Immediate
		pubMsg,       // Message
	)
}

// Consume will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (client *Client) Consume() (<-chan amqp.Delivery, error) {
	if !client.isReady {
		return nil, errNotConnected
	}
	return client.channel.Consume(
		client.queueName,
		client.name, // Consumer
		false,       // Auto-Ack
		false,       // Exclusive
		false,       // No-local
		false,       // No-Wait
		nil,         // Args
	)
}

// Close will cleanly shutdown the channel and connection.
func (client *Client) Close() error {
	if !client.isReady {
		return errAlreadyClosed
	}
	close(client.done)
	err := client.channel.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	err = client.connection.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	client.isReady = false
	return nil
}
