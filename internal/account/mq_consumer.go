package account

import (
	"fmt"
	"log"
	"time"
	"xmpp/internal/pkg/amqp/rabbitmq"
	"xmpp/internal/server"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m Management) ConsumeRoutine(consumeExchange, consumerName, consumerQueueName, url string) {
	client := rabbitmq.NewClient(consumerName, consumerQueueName, url)
	log.Println("consumeExchange: ", consumeExchange)
	log.Println("consumerName: ", consumerName)
	log.Println("consumerQueueName: ", consumerQueueName)
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

		log.Println("wait to send message from publisher")

		for msg := range messages {

			go func(inMsg amqp.Delivery) {
				method := ""
				if inMsg.Headers["method"] != nil {
					method = inMsg.Headers["method"].(string)
				}

				switch method {
				case "connreq":
					endpointID := ""
					if inMsg.Headers["endPointID"] != nil {
						endpointID = inMsg.Headers["endPointID"].(string)
					}

					taskId := ""
					topicId := ""
					if inMsg.Headers["taskId"] != nil {
						taskId = inMsg.Headers["taskId"].(string)
					}
					if inMsg.Headers["topicId"] != nil {
						topicId = inMsg.Headers["topicId"].(string)
					}

					toJid := fmt.Sprintf("%v@%v/%v", endpointID, m.AdminUser.Domain, m.AdminUser.Resource)
					fromJid := fmt.Sprintf("%v@%v/%v", m.AdminUser.Name, m.AdminUser.Domain, m.AdminUser.Resource)
					pushMsg := server.ConnectionRequest{
						TaskId:      taskId,
						TopicId:     topicId,
						FromJid:     fromJid,
						ToJid:       toJid,
						ToLocalPart: endpointID,
						Password:    m.AdminUser.Password,
					}
					m.Mutex.Lock()
					if channel, ok := m.Online[toJid]; ok {
						channel <- pushMsg
					} else {
						// nok
						// publish cr error
						// publish offline
					}
					m.Mutex.Unlock()
				}
				inMsg.Ack(true)
			}(msg)

		}
	}
}
