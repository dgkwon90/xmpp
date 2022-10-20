package server

import (
	"encoding/xml"
	"log"
	"strings"
)

// Extension interface for processing normal messages
type Extension interface {
	Process(message interface{}, from *Client)
}

// DebugExtension just dumps data
type DebugExtension struct {
	Log Logging
}

// Process a message (write to debug logger)
func (e *DebugExtension) Process(message interface{}, from *Client) {
	data, _ := xml.Marshal(message)
	//e.Log.Debug("Processing message: " + string(data))
	log.Println("[ex][DebugExtension] Processing message: " + string(data))
}

// NormalMessageExtension handles client messages
type NormalMessageExtension struct {
	MessageBus chan<- Message
}

// Process sends `ClientMessage`s from a client down the `MessageBus`
func (e *NormalMessageExtension) Process(message interface{}, from *Client) {
	parsed, ok := message.(*ClientMessage)
	if ok {
		log.Println("[ex][NormalMessageExtension] NormalMessageExtension Process")
		e.MessageBus <- Message{To: parsed.To, Data: message}
		log.Printf("[ex][NormalMessageExtension] to: %v, data: %v\n", parsed.To, message)
	}
}

// RosterExtension handles ClientIQ presence requests and updates
type RosterExtension struct {
	Accounts AccountManager
}

// Process responds to Presence requests from a client
func (e *RosterExtension) Process(message interface{}, from *Client) {
	parsed, ok := message.(*ClientIQ)

	// handle things we need to handle
	if ok {
		log.Printf("[ex] type: %v\n", parsed.Type)
		log.Printf("[ex] From: %v\n", parsed.From)
		log.Printf("[ex] To: %v\n", parsed.To)
		log.Printf("[ex] Error: %v\n", parsed.Error)
		log.Printf("[ex] Bind: %v\n", parsed.Bind)
		log.Printf("[ex] ID: %v\n", parsed.ID)
		log.Printf("[ex] XMLName: %v\n", parsed.XMLName)
		log.Printf("[ex] Query: %v\n", string(parsed.Query))
		log.Printf("[ex] ConnectionRequest: %v\n", parsed.ConnectionRequest)

		// // Receive ping from client. *reference XEP-0199: XMPP Ping
		// //if parsed.Type == "get" && (string(parsed.Query) == "<ping xmlns=\"urn:xmpp:ping\"/>") {
		// if parsed.Type == "get" && parsed.ID == "c2s1" {
		// 	log.Println("[ex] receive ping")
		// 	// response pong message
		// 	msg := fmt.Sprintf("<iq from='%v' to='%v' id='c2s1' type='result'/>", parsed.To, parsed.From)
		// 	from.messages <- msg
		if parsed.ID == "cr001" && parsed.Type == "get" {
			//crMap := new(map[string]interface{})
			//xmlErr := xml.Unmarshal(parsed.Query, crMap)
			//if xmlErr != nil {
			//	log.Println("[ex] query xml parsing err: ", xmlErr)
			//} else {
			//	log.Println("[ex] query xml: ", crMap)
			//}
			log.Printf("[ex] ConnectionRequest get")
			log.Printf("[ex] ConnectionRequest.Username: %v\n", parsed.ConnectionRequest.UserName)
			log.Printf("[ex] ConnectionRequest.Password: %v\n", parsed.ConnectionRequest.Password)
			crMsg := CreateConnectionRequest("","", parsed.To, parsed.From, parsed.ConnectionRequest.UserName, parsed.ConnectionRequest.Password)
			e.Accounts.SendConnectionRequest(crMsg)
		}

		if parsed.ID == "cr001" && parsed.Type == "result" {
			log.Printf("[ex] ConnectionRequest")
			log.Printf("[ex] ConnectionRequest from.jid: %v\n", from.jid)
			log.Printf("[ex] ConnectionRequest from.localPart: %v\n", from.localPart)
			e.Accounts.ConnectionRequestResult(from.jid, from.localPart, true)
		}

		if string(parsed.Query) == "<error type='wait'><resource-constraint/></error>" ||
			string(parsed.Query) == "<error type=\"wait\"><resource-constraint/></error>" {
			log.Printf("[ex] Device Busy Try again : %v", from.jid)
			e.Accounts.ConnectionRequestResult(from.jid, from.localPart, false)
			//go func() {
			//	time.Sleep(time.Second * 10)
			//	from.messages <- "connreq"
			//}()
		}

		if string(parsed.Query) == "<query xmlns='jabber:iq:roster'/>" {
			// respond with roster
			roster, _ := e.Accounts.OnlineRoster(from.jid)
			msg := "<iq id='" + parsed.ID + "' to='" + parsed.From + "' type='result'><query xmlns='jabber:iq:roster' ver='ver7'>"
			for _, v := range roster {
				msg = msg + "<item jid='" + v + "'/>"
			}
			msg = msg + "</query></iq>"

			// respond to client
			from.messages <- msg
		}

		if parsed.Type == "set" && (string(parsed.Query) == "<session xmlns=\"urn:ietf:params:xml:ns:xmpp-session\"/>" ||
			string(parsed.Query) == "<session xmlns='urn:ietf:params:xml:ns:xmpp-session'/>") {
			ids := strings.Split(from.jid, "/")
			msg := "<iq xml:lang='en' to='" + from.jid + "' from='" + ids[0] + "' type='result' id='_xmpp_session1'/>"
			from.messages <- msg
		}
	} else {
		log.Println("[ex] no query")
	}
}

// PresenceExtension handles ClientIQ presence requests and updates
type PresenceExtension struct {
	PresenceBus chan<- Message
}

// Process responds to Presence message from a client
func (e *PresenceExtension) Process(message interface{}, from *Client) {
	parsed, ok := message.(*ClientPresence)
	if ok {
		log.Printf("[ex] presence extension: %v", parsed)
		log.Printf("[ex] delay: %v", parsed.Delay)
		// this is how you reply to a message: from.messages <- message

		// types:
		// subscribe
		// subscribed
		// unsubscribed

		// 1) check request and remove Resource

		// 2) put request on the presence bus
		// 3) put roster push on message bus
		// or
		// auto reply with 'subscribed' if invite in other direction

		// compose the presence message (from) put on a channel that
		// the server will then push to the message chan for all other clients
		// difficult to - alert the 'to' field of the message

		// if i receive a presence message from the client, put it on the presence
		// bus for broadcasting to subscribers/peers
		// server should alter message
		log.Println("[ex] To: ", parsed.To)
		log.Println("[ex] Msg: ", message)
		if len(parsed.To) > 0 {
			e.PresenceBus <- Message{To: parsed.To, Data: message}
		} else {
			log.Println("[ex] To is nil: ", parsed.To)
		}
	} else {
		log.Println("[ex] no presence")
	}
}
