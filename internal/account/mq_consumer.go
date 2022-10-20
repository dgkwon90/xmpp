package account

import (
	"fmt"
	"log"
	"time"
	"xmpp/internal/pkg/amqp/rabbitmq"
	"xmpp/internal/server"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m Management) ConsumeRoutine(consumeExchange, consumerName, consumerQueueName, url, envServerClient, domain, resource, password string) {
	client := rabbitmq.NewClient(consumerName, consumerQueueName, url)
	log.Println("[am][mq] consumeExchange: ", consumeExchange)
	log.Println("[am][mq] consumerName: ", consumerName)
	log.Println("[am][mq] consumerQueueName: ", consumerQueueName)
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

		log.Println("[am][mq] wait to send message from publisher")

		for msg := range messages {
			go func(inMsg amqp.Delivery) {
				log.Printf("[am][mq] receive msg: %v\n", inMsg)
				method := ""
				if inMsg.Headers["method"] != nil {
					method = inMsg.Headers["method"].(string)
				}

				switch method {
				case "connreq":
					log.Println("[am][mq] method is connreq")
					endpointID := ""
					if inMsg.Headers["endPointID"] != nil {
						endpointID = inMsg.Headers["endPointID"].(string)
					}

					taskId := ""
					topicId := ""
					if inMsg.Headers["taskId"] != nil {
						taskId = inMsg.Headers["taskId"].(string)
					}
					log.Printf("[am][mq] taskId: %v\n", taskId)

					if inMsg.Headers["topicId"] != nil {
						topicId = inMsg.Headers["topicId"].(string)
					}
					log.Printf("[am][mq] topicId: %v\n", topicId)

					toJid := fmt.Sprintf("%v@%v/%v", envServerClient, domain, resource)
					fromJid := fmt.Sprintf("%v@%v/%v", endpointID, domain, resource)

					log.Printf("[am][mq] toJid: %v\n", toJid)
					log.Printf("[am][mq] fromJid: %v\n", fromJid)

					crMsg := server.CreateConnectionRequest(taskId, topicId, fromJid, toJid, endpointID, password)
					m.SendConnectionRequest(crMsg)
				}
				inMsg.Ack(true)
			}(msg)

		}
	}
}
