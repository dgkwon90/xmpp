package consumer

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer 구조체
type Consumer struct {
	Url        string
	Name       string
	Exchange   string
	QueueName  string
	RoutingKey string
	Headers    map[string]interface{}
	// session
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// 수신 받은 메시지를 처리하는 Handler 함수
type ReciveMsgHandler func(name string, msg interface{})

// New 새로운 Consumer를 생성
func New(url, name, exchange, queueName, routingKey string, headers map[string]interface{}) *Consumer {
	con := new(Consumer)
	con.Url = url
	con.Name = name
	con.Exchange = exchange
	con.QueueName = queueName
	con.RoutingKey = routingKey
	con.Headers = headers
	fmt.Printf("[%v] New Consumer :%v\n", con.Name, con)
	return con
}

// Connection Broker와 연결
func (c *Consumer) Connection() error {
	conn, connErr := amqp.Dial(c.Url)
	if connErr != nil {
		fmt.Printf("[%v] Rabbit MQ Connection Fail %v\n", c.Name, connErr.Error())
		return connErr
	}

	c.Conn = conn
	return nil
}

// OpenChannel Channel을 연다
func (c *Consumer) OpenChannel() error {
	ch, chErr := c.Conn.Channel()
	if chErr != nil {
		fmt.Printf("[%v] Channel Fail %v\n", c.Name, chErr.Error())
		return chErr
	}
	c.Channel = ch
	return nil
}

// Close Channel, Connection을 Close 수행
func (c *Consumer) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.Conn != nil {
		c.Conn.Close()
	}
}

// Bind Exchange 생성, Queue 생성, Queue Bindig 후 Consumer Queue Bind 수행
func (c *Consumer) Bind(exchangeType string, handler ReciveMsgHandler) error {
	exchangeDeclareErr := c.Channel.ExchangeDeclare(
		c.Exchange,   // name
		exchangeType, // type
		false,        // durable
		true,         // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if exchangeDeclareErr != nil {
		fmt.Printf("[%v] ExchangeDeclare Error %v\n", c.Name, exchangeDeclareErr)
		return exchangeDeclareErr
	}

	_, queueDeclareErr := c.Channel.QueueDeclare(
		c.QueueName, // name
		false,       // durable
		true,        // auto-deleted
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if queueDeclareErr != nil {
		fmt.Printf("[%v] QueueDeclare Error %v\n", c.Name, queueDeclareErr)
		return queueDeclareErr
	}

	bindErr := c.Channel.QueueBind(
		c.QueueName,  // name
		c.RoutingKey, // key(routing)
		c.Exchange,   // exchange
		false,        // no-wait
		c.Headers,    // arguments
	)
	if bindErr != nil {
		fmt.Printf("[%v] QueueBind Error %v\n", c.Name, bindErr)
		return bindErr
	}

	messages, consumeErr := c.Channel.Consume(
		c.QueueName, // queue
		c.Name,      // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if consumeErr != nil {
		fmt.Printf("[%v] Consume Error %v\n", c.Name, consumeErr)
		return consumeErr
	}

	fmt.Printf("[%v] Start wait message...\n", c.Name)
	for msg := range messages {
		handler(c.Name, msg)
		msg.Ack(true)
	}
	return nil
}
