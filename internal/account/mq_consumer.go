package account

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
	"xmpp/internal/pkg/amqp/rabbitmq"
)

func (m Management) ConsumeRoutine(consumeExchange, consumerName, consumerQueueName, url string) {
	client := rabbitmq.NewClient(consumerName, consumerQueueName, url)
	for {
		if !client.IsReady() {
			time.Sleep(rabbitmq.ResendDelay)
			continue
		}

		err := client.ExchangeDeclare(consumeExchange, "fanout")
		if err != nil {
			log.Printf("ExchangeDeclare Error: %v\n", err)
			continue
		}


		err = client.QueueBind(consumeExchange, "")
		if err != nil {
			log.Printf("QueueBind Error: %v\n", err)
			continue
		}

		messages, consumeErr := client.Consume()
		if consumeErr != nil {
			log.Printf("Consume Error :%v\n", consumeErr)
			continue
		}

		if messages == nil {
			log.Println("messages is nil")
			continue
		}

		for msg := range messages {
			go func(inMsg amqp.Delivery) {
				endpointID := ""
				method := ""
				taskId := ""
				taskTopicId := ""
				if inMsg.Headers["endPointID"] != nil {
					endpointID = inMsg.Headers["endPointID"].(string)
				}
				if inMsg.Headers["method"] != nil {
					method = inMsg.Headers["method"].(string)
				}
				if taskId != inMsg.Headers["taskId"] {
					taskId = inMsg.Headers["taskId"].(string)
				}
				if taskTopicId != inMsg.Headers["taskTopicId"] {
					taskTopicId = inMsg.Headers["taskTopicId"].(string)
				}

			}(msg)
		}
	}
}
