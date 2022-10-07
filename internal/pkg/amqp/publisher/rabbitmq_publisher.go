package publisher

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher 구조체
type Publisher struct {
	Url  string
	Name string
	// session
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// New 새로운 Publisher를 생성
func New(url, name string) *Publisher {
	pub := new(Publisher)
	pub.Url = url
	pub.Name = name
	fmt.Printf("[%v] New Publisher :%v\n", pub.Name, pub)
	return pub
}

// Connection Broker와 연결
func (p *Publisher) Connection() error {
	conn, connErr := amqp.Dial(p.Url)
	if connErr != nil {
		fmt.Printf("[%v] Rabbit MQ Connection Fail %v\n", p.Name, connErr.Error())
		return connErr
	}

	p.Conn = conn
	return nil
}

// OpenChannel Channel을 연다
func (p *Publisher) OpenChannel() error {
	ch, chErr := p.Conn.Channel()
	if chErr != nil {
		fmt.Printf("[%v] Channel Fail %v\n", p.Name, chErr.Error())
		return chErr
	}
	p.Channel = ch
	return nil
}

// Close Channel, Connection을 Close 수행
func (p *Publisher) Close() {
	if p.Channel != nil {
		p.Channel.Close()
	}
	if p.Conn != nil {
		p.Conn.Close()
	}
}

// Publish 메세지를 broker에 전송
func (p *Publisher) Publish(exchangeName string, routingKey string, mandatory, immediate bool, pubMsg amqp.Publishing) error {
	publishErr := p.Channel.PublishWithContext(
		context.Background(), // context
		exchangeName,         // exchange
		routingKey,           // routing key
		mandatory,            // mandatory
		immediate,            // immediate
		pubMsg)
	if publishErr != nil {
		fmt.Printf("[%v] publish Error %v\n", p.Name, publishErr)
		return publishErr
	}
	fmt.Printf("[%v] push message: [%v] %v  => \n", p.Name, routingKey, pubMsg.MessageId)
	return nil
}