package account

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (m Management) notifyConnectionStatus(exchangeName, endpoint, connectionStatus string) {
	if m.UseMQ {
		mapBody := map[string]interface{}{
			"endpoint_id":       endpoint,
			"connection_status": connectionStatus,
		}
		jsonBody, marshalErr := json.Marshal(mapBody)
		if marshalErr != nil {
			log.Println("jsonMarshalErr: ", marshalErr)
		} else {
			msg := amqp.Publishing{
				ContentType: "Application/json",
				//MessageId:   "Msg1",
				Headers: nil,
				Body:    jsonBody,
			}

			m.Publisher.Push(
				exchangeName,
				"",
				false,
				false,
				msg,
			)
			log.Println("[am][mq] Push Device Event")
		}
	}
}

func (m Management) notifyConnectionRequest(exchangeId, taskId, endpointId, reason string) {
	if m.UseMQ {
		mapBody := map[string]interface{}{
			"Id":   taskId,
			"From": endpointId,
			"To":   exchangeId,
		}
		if len(reason) > 0 {
			mapBody["Error"] = reason
		}

		jsonBody, marshalErr := json.Marshal(mapBody)
		if marshalErr != nil {
			log.Println("jsonMarshalErr: ", marshalErr)
		} else {
			msg := amqp.Publishing{
				ContentType: "Application/json",
				//MessageId:   "Msg1",
				Headers: nil,
				Body:    jsonBody,
			}

			m.Publisher.Push(
				exchangeId,
				"",
				false,
				false,
				msg,
			)
			log.Println("[am][mq] Push Device Event")
		}
	}
}
