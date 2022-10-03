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
	//Command  chan<- interface{}
}

//Management Inject account management into xmpp library
type Management struct {
	AdminUser AdminUser
	//Users     map[string]string
	Online map[string]chan<- interface{}
	Mutex  *sync.Mutex
	Log    logger.Logger
}

func (a Management) Authenticate(username, password string) (success bool, err error) {
	//a.log.Info("start authenticate")
	log.Println("[am] >>>> start authenticate")

	//a.log.Info(fmt.Sprintf("authenticate: %s", username))
	log.Printf("[am] >>>> authenticate: %s\n", username)

	//if _, ok := a.Users[username]; ok {
		// Already created user(client)
		if a.AdminUser.Password == password {
			//Continue the current state.
			//a.log.Debug("auth success")
			log.Println("[am] >>>> auth success")
			success = true
		} else {
			//a.Mutex.Lock()
			//defer a.Mutex.Unlock()

			//Auth fail, delete user.
			//a.DeleteAccount(username)

			//a.log.Debug("auth fail")
			log.Println("[am] >>>> auth fail")
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
	log.Printf("[am] >>>> retrieving roster: %v\n", jid)

	for person := range a.Online {
		online = append(online, person)
	}
	return
}

// PresenceRoutine new WIP func for presence messages
func (a Management) PresenceRoutine(bus <-chan server.Message) {
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
	for {
		message := <-bus
		a.Mutex.Lock()
		//a.log.Info(fmt.Sprintf("[am] %s disconnected", message.Jid))
		log.Printf("[am] %v disconnected\n", message.Jid)
		delete(a.Online, message.Jid)
		a.Mutex.Unlock()
	}
}