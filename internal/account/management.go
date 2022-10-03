package account

import (
	"log"
	"sync"
	"xmpp/internal/pkg/logger"
	"xmpp/internal/server"
)

type AdminUser struct {
	Name     string
	Password string
	Resource string
	//Command  chan<- interface{}
}

// Online Heartbeat
type Online struct {
	Receiver map[string]chan<- interface{}
	// Heartbeat
	HeartBeatNumberOfDelay int
}

// Management Inject account management into xmpp library
type Management struct {
	AdminUser AdminUser
	//Users     map[string]string
	Online map[string]chan<- interface{}
	Mutex  *sync.Mutex
	Log    logger.Logger

	// Heartbeat
	//HeartBeatMaxNumberOfDelay int // Default 3
	//HeartbeatRateIntervalSeconds int // Default 35
}

func (a Management) Authenticate(username, password string) (success bool, err error) {
	//a.log.Info("start authenticate")
	log.Println("[am] start authenticate")

	//a.log.Info(fmt.Sprintf("authenticate: %s", username))
	log.Printf("[am] authenticate name: %s\n", username)
	log.Printf("[am] authenticate password: %s\n", password)

	//if _, ok := a.Users[username]; ok {
	// Already created user(client)
	if a.AdminUser.Password == password {
		//Continue the current state.
		//a.log.Debug("auth success")
		log.Println("[am] authenticate success")
		success = true
	} else {
		//a.Mutex.Lock()
		//defer a.Mutex.Unlock()

		//Auth fail, delete user.
		//a.DeleteAccount(username)

		//a.log.Debug("auth fail")
		log.Println("[am] authenticate fail")
		success = false
	}
	//} else {
	//	log.Println("[am] >>>> new username")
	//	success, err = a.CreateAccount(username, password)
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

func (a Management) OnlineRoster(jid string) (online []string, err error) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	//a.log.Info(fmt.Sprintf("retrieving roster: %s", jid))
	log.Printf("[am] retrieving roster: %v\n", jid)

	for person := range a.Online {
		online = append(online, person)
	}
	return
}

// PresenceRoutine new WIP func for presence messages
func (a Management) PresenceRoutine(bus <-chan server.Message) {
	log.Println("[am] new rresenceroutine")

	for {
		message := <-bus
		a.Mutex.Lock()

		for _, userChannel := range a.Online {
			userChannel <- message.Data
		}

		a.Mutex.Unlock()
	}
}

func (a Management) RouteRoutine(bus <-chan server.Message) {
	var channel chan<- interface{}
	var ok bool

	log.Println("[am] new route routine")
	for {
		message := <-bus
		a.Mutex.Lock()

		if channel, ok = a.Online[message.To]; ok {
			channel <- message.Data
		}

		a.Mutex.Unlock()
	}
}

func (a Management) ConnectRoutine(bus <-chan server.Connect) {
	log.Println("[am] new connect routine")
	for {
		message := <-bus
		a.Mutex.Lock()

		//a.log.Info(fmt.Sprintf("[am] %s connected", message.Jid))
		log.Printf("[am] %v connected\n", message.Jid)
		a.Online[message.Jid] = message.Receiver

		a.Mutex.Unlock()
	}
}

func (a Management) DisconnectRoutine(bus <-chan server.Disconnect) {
	log.Println("[am] new disconnect routine")
	for {
		message := <-bus
		a.Mutex.Lock()

		//a.log.Info(fmt.Sprintf("[am] %s disconnected", message.Jid))
		log.Printf("[am] %v disconnected\n", message.Jid)
		delete(a.Online, message.Jid)

		a.Mutex.Unlock()
	}
}
