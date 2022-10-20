package account

import (
	"context"
	"log"
	"sync"
	"time"
	"xmpp/internal/pkg/amqp/rabbitmq"
	"xmpp/internal/pkg/logger"
	"xmpp/internal/pkg/memstore"
	"xmpp/internal/server"
)

type AdminUser struct {
	Name     string
	Password string
	//Domain   string
	//Resource string
	//Command  chan<- interface{}
}

// Management Inject account management into xmpp library
type Management struct {
	AdminUser    AdminUser
	SkipPassword bool
	//Users     map[string]string
	Online map[string]chan<- interface{}

	ConnectionRequests map[string]chan bool

	Mutex *sync.Mutex
	Log   logger.Logger

	// redis
	UseDB  bool
	DBType string
	DB     memstore.Redis

	// rabbitmq
	UseMQ     bool
	MQType    string
	Publisher *rabbitmq.Client
	//Consumer *rabbitmq.Client
	ID string
}

func (m Management) Authenticate(username, password string) (success bool, err error) {
	//m.log.Info("start authenticate")
	log.Println("[am] start authenticate")

	//m.log.Info(fmt.Sprintf("authenticate: %s", username))
	log.Printf("[am] authenticate name: %s\n", username)
	log.Printf("[am] authenticate password: %s\n", password)

	if m.SkipPassword {
		log.Println("[am] skip password true!!! authenticate success")
		success = true
		return
	}

	//if _, ok := m.Users[username]; ok {
	// Already created user(client)
	if m.AdminUser.Password == password {
		//Continue the current state.
		//m.log.Debug("auth success")
		log.Println("[am] authenticate success")
		success = true
	} else {
		//m.Mutex.Lock()
		//defer m.Mutex.Unlock()

		//Auth fail, delete user.
		//m.DeleteAccount(username)

		//m.log.Debug("auth fail")
		log.Println("[am] authenticate fail")
		success = false
	}
	//} else {
	//	log.Println("[am] >>>> new username")
	//	success, err = m.CreateAccount(username, password)
	//}
	return
}

//func (a Management) CreateAccount(username, password string) (success bool, err error) {
//	a.Mutex.Lock()
//	defer a.Mutex.Unlock()
//
//	//a.log.Info(fmt.Sprintf("create account: %s", username))
//	log.Printf("[am] >>>> create account: %v\n", username)
//
//	if _, err := a.Users[username]; err {
//		success = false
//	} else {
//		a.Users[username] = password
//		success = true
//	}
//	return
//}

//func (a Management) DeleteAccount(username string){
//	a.Mutex.Lock()
//	defer a.Mutex.Unlock()
//
//	//a.log.Info(fmt.Sprintf("create account: %s", username))
//	log.Printf("[am] >>>> delete account: %v\n", username)
//	delete(a.Users, username)
//}

func (m Management) OnlineRoster(jid string) (online []string, err error) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	//m.log.Info(fmt.Sprintf("retrieving roster: %s", jid))
	log.Printf("[am] retrieving roster: %v\n", jid)

	for person := range m.Online {
		online = append(online, person)
	}
	return
}

func (m Management) SendConnectionRequest(message server.ConnectionRequest) {
	log.Println("[am] SendConnectionRequest ")
	channel, ok := m.Online[message.ToJid]
	if ok {
		log.Printf("[am] SendConnectionRequest tojid: %v, fromjid: %v\n", message.ToJid, message.FromJid)
		m.Mutex.Lock()
		channel <- message
		m.Mutex.Unlock()
	} else {
		log.Printf("[am] ConnectionRequests tojid:%v is null\n", message.ToJid)
		// nok
		// publish cr error
		// publish offline
	}
}

// PresenceRoutine new WIP func for presence messages
func (m Management) PresenceRoutine(bus <-chan server.Message) {
	log.Println("[am] new presence routine")

	for {
		message := <-bus
		m.Mutex.Lock()
		//message.To
		for jid, userChannel := range m.Online {
			log.Println("[am] jid: ", jid)
			if userChannel != nil {
				userChannel <- message.Data
			} else {
				log.Println("[am] user channel is nil")
			}
		}
		m.Mutex.Unlock()
	}
}

func (m Management) RouteRoutine(bus <-chan server.Message) {
	var channel chan<- interface{}
	var ok bool

	log.Println("[am] new route routine")
	for {
		message := <-bus
		m.Mutex.Lock()

		if channel, ok = m.Online[message.To]; ok {
			channel <- message.Data
		}

		m.Mutex.Unlock()
	}
}

func (m Management) ConnectRoutine(bus <-chan server.Connect) {
	log.Println("[am] new connect routine")
	for {
		message := <-bus
		m.Mutex.Lock()

		//m.log.Info(fmt.Sprintf("[am] %s connected", message.Jid))
		log.Printf("[am] %v connected\n", message.Jid)
		m.Online[message.Jid] = message.Receiver

		m.Mutex.Unlock()

		// TODO: Redis Set Or Update Online
		key := message.LocalPart
		val := m.ID
		m.saveOnline(key, val)

		// TODO: Push Event
		m.notifyConnectionStatus("device-connection-topic", key, "online")
	}
}

func (m Management) DisconnectRoutine(bus <-chan server.Disconnect) {
	log.Println("[am] new disconnect routine")
	for {
		message := <-bus
		m.Mutex.Lock()

		//m.log.Info(fmt.Sprintf("[am] %s disconnected", message.Jid))
		log.Printf("[am] %v disconnected\n", message.Jid)
		delete(m.Online, message.Jid)
		m.Mutex.Unlock()

		// TODO: Redis Set Or Update Offline
		key := message.LocalPart
		m.saveOffline(key)

		// TODO: Push Event
		m.notifyConnectionStatus("device-connection-topic", key, "offline")
	}
}

func (m Management) ConnectionRequestRoutine(bus <-chan server.ConnectionRequest) {
	log.Println("[am] new connection Request routine")
	for {
		message := <-bus
		m.Mutex.Lock()
		//m.log.Info(fmt.Sprintf("[am] %s connected", message.Jid))
		log.Printf("[am] %v Connection Request\n", message.ToJid)
		m.ConnectionRequests[message.ToJid] = make(chan bool, 1)
		m.Mutex.Unlock()
		go m.connectionRequestResultRoutine(message, 10)
	}
}

func (m Management) connectionRequestResultRoutine(msg server.ConnectionRequest, timeoutSec time.Duration) {
	if _, ok := m.ConnectionRequests[msg.ToJid]; ok {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeoutSec)
		defer cancel()

		log.Printf("[am] wait connection reqeust of %v\n", msg.ToLocalPart)
		select {
		case result := <- m.ConnectionRequests[msg.ToJid]:
			if result {
				// success
				log.Printf("[am] connectionRequestResultRoutine [%v]: success\n", msg.ToJid)
				log.Println("[am] connectionRequestResultRoutine success: ", msg.ToLocalPart)

				m.Mutex.Lock()
				delete(m.ConnectionRequests, msg.ToJid)
				m.Mutex.Unlock()

				if len(msg.TaskId) > 0 {
					// response server(rabbitmq)
					m.notifyConnectionRequest(msg.TopicId, msg.TaskId, msg.ToLocalPart, "")
				} else {
					// response client(xmpp)
					// TODO: client to xmpp error
				}
			} else {
				// fail
				log.Printf("[am] connectionRequestResultRoutine [%v]: fail busy(%v)\n", msg.ToJid, timeoutSec)
				log.Println("[am] connectionRequestResultRoutine fail: ", msg.ToLocalPart)

				m.Mutex.Lock()
				delete(m.ConnectionRequests, msg.ToJid)
				m.Mutex.Unlock()

				reason := "busy"
				if len(msg.TaskId) > 0 {
					// response server(rabbitmq)
					m.notifyConnectionRequest(msg.TopicId, msg.TaskId, msg.ToLocalPart, reason)
				} else {
					// response client(xmpp)
					// TODO: client to xmpp error
				}
			}
		case <-ctx.Done():
			// timeout
			log.Printf("[am] connectionRequestResultRoutine [%v]: fail timeout(%v)\n", msg.ToJid, timeoutSec)
			log.Println("[am] connectionRequestResultRoutine timeout: ", msg.ToLocalPart)

			m.Mutex.Lock()
			delete(m.ConnectionRequests, msg.ToJid)
			reason := "timeout"
			if len(msg.TaskId) > 0 {
				// response server(rabbitmq)
				m.notifyConnectionRequest(msg.TopicId, msg.TaskId, msg.ToLocalPart, reason)
			} else {
				// response client(xmpp)
				// TODO: client to xmpp error
			}
		}

	} else {
		// error
		log.Printf("[am] ConnectionRequestResultRoutine [%v]: fail no chan\n", msg.ToJid)
	}
}

func (m Management) ConnectionRequestResult(jid, localPart string, result bool) {
	log.Printf("[am] ConnectionRequestResult [%v][%v]: %v\n", jid, localPart, result)
	if _, ok := m.ConnectionRequests[jid];ok {
		log.Printf("[am] ConnectionRequestResult is chan[%v]\n", jid)
		m.Mutex.Lock()
		m.ConnectionRequests[jid] <- result
		m.Mutex.Unlock()
	} else {
		log.Printf("[am] ConnectionRequestResult [%v]: fail no chan\n", jid)
	}
}
