package account

import (
	"log"
	"sync"
	"xmpp/internal/pkg/logger"
	"xmpp/internal/pkg/memstore"
	"xmpp/internal/server"
)

type AdminUser struct {
	Name     string
	Password string
	Resource string
	//Command  chan<- interface{}
}

// Management Inject account management into xmpp library
type Management struct {
	AdminUser    AdminUser
	SkipPassword bool
	//Users     map[string]string
	Online map[string]chan<- interface{}
	Mutex  *sync.Mutex
	Log    logger.Logger

	// redis
	UseRedis bool
	Redis    memstore.Redis

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
	}
}
