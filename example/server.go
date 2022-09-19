package main

import (
	"log"

	"xmpp"

	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
)

/* Inject logging into xmpp library */

type Logger struct {
	info  bool
	debug bool
}

func (l Logger) Info(msg string) (err error) {
	if l.info {
		//_, err = fmt.Printf("INFO: %s\n", msg)
		log.Printf("INFO: %s\n", msg)
	}
	return err
}

func (l Logger) Debug(msg string) (err error) {
	if l.debug {
		//_, err = fmt.Printf("DEBUG: %s\n", msg)
		log.Printf("DEBUG: %s\n", msg)
	}
	return err
}

func (l Logger) Error(msg string) (err error) {
	//_, err = fmt.Printf("ERROR: %s\n", msg)
	log.Printf("ERROR: %s\n", msg)
	return err
}

/* Inject account management into xmpp library */

type AccountManager struct {
	Users  map[string]string
	Online map[string]chan<- interface{}
	lock   *sync.Mutex
	log    Logger
}

func (a AccountManager) Authenticate(username, password string) (success bool, err error) {
	//a.log.Info("start authenticate")
	log.Println("start authenticate")
	a.lock.Lock()
	defer a.lock.Unlock()

	//a.log.Info(fmt.Sprintf("authenticate: %s", username))
	log.Printf("authenticate: %s\n", username)
	if a.Users["krms-server"] == password {
		//a.log.Debug("auth success")
		log.Println("auth success")
		success = true
	} else {
		//a.log.Debug("auth fail")
		log.Println("auth fail")
		success = false
	}

	return
}

func (a AccountManager) CreateAccount(username, password string) (success bool, err error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	//a.log.Info(fmt.Sprintf("create account: %s", username))
	log.Printf("create account: %v\n", username)

	if _, err := a.Users[username]; err {
		success = false
	} else {
		a.Users[username] = password
	}
	return
}

func (a AccountManager) OnlineRoster(jid string) (online []string, err error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	//a.log.Info(fmt.Sprintf("retrieving roster: %s", jid))
	log.Printf("retrieving roster: %v\n", jid)

	for person := range a.Online {
		online = append(online, person)
	}
	return
}

// new WIP func for pressence messages
func (a AccountManager) presenceRoutine(bus <-chan xmpp.Message) {
	for {
		message := <-bus
		a.lock.Lock()

		for _, userChannel := range a.Online {
			userChannel <- message.Data
		}

		a.lock.Unlock()
	}
}

func (a AccountManager) routeRoutine(bus <-chan xmpp.Message) {
	var channel chan<- interface{}
	var ok bool

	for {
		message := <-bus
		a.lock.Lock()

		if channel, ok = a.Online[message.To]; ok {
			channel <- message.Data
		}

		a.lock.Unlock()
	}
}

func (a AccountManager) connectRoutine(bus <-chan xmpp.Connect) {
	for {
		message := <-bus
		a.lock.Lock()
		//a.log.Info(fmt.Sprintf("[am] %s connected", message.Jid))
		log.Printf("[am] %v connected\n", message.Jid)
		a.Online[message.Jid] = message.Receiver
		a.lock.Unlock()
	}
}

func (a AccountManager) disconnectRoutine(bus <-chan xmpp.Disconnect) {
	for {
		message := <-bus
		a.lock.Lock()
		//a.log.Info(fmt.Sprintf("[am] %s disconnected", message.Jid))
		log.Printf("[am] %v disconnected\n", message.Jid)
		delete(a.Online, message.Jid)
		a.lock.Unlock()
	}
}

/* Main server loop */

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	envPort := 5222
	envDebug := true
	envSkipTLS := true
	envDomian := "localhost"
	envAdmin := "krms-server"
	envPass := "test1234"

	portPtr := flag.Int("port", envPort, "port number to listen on")
	debugPtr := flag.Bool("debug", envDebug, "turn on debug logging")
	flag.Parse()

	var registered = make(map[string]string)
	registered[envAdmin] = envPass

	var activeUsers = make(map[string]chan<- interface{})

	var l = Logger{info: true, debug: *debugPtr}

	var messagebus = make(chan xmpp.Message)
	var presencebus = make(chan xmpp.Message)
	var connectbus = make(chan xmpp.Connect)
	var disconnectbus = make(chan xmpp.Disconnect)

	var am = AccountManager{Users: registered, Online: activeUsers, log: l, lock: &sync.Mutex{}}

	var cert, _ = tls.LoadX509KeyPair("./cert.pem", "./key.pem")
	var tlsConfig = tls.Config{
		MinVersion:   tls.VersionTLS10,
		Certificates: []tls.Certificate{cert},
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA},
	}

	xmppServer := &xmpp.Server{
		SkipTLS:    envSkipTLS,
		Log:        l,
		Accounts:   am,
		ConnectBus: connectbus,
		Extensions: []xmpp.Extension{
			&xmpp.DebugExtension{Log: l},
			&xmpp.NormalMessageExtension{MessageBus: messagebus},
			&xmpp.RosterExtension{Accounts: am},
			&xmpp.PresenceExtension{PresenceBus: presencebus},
		},
		DisconnectBus: disconnectbus,
		Domain:        envDomian,
		TLSConfig:     &tlsConfig,
	}

	// l.Info("Starting server")
	log.Println("Starting server")
	// l.Info("Listening on localhost:" + fmt.Sprintf("%d", *portPtr))
	log.Printf("Listening on localhost: %v\n", *portPtr)

	// Listen for incoming connections.

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *portPtr))
	if err != nil {
		l.Error(fmt.Sprintf("Could not listen for connections: %s", err.Error()))
		os.Exit(1)
	}
	defer listener.Close()

	go am.routeRoutine(messagebus)
	go am.connectRoutine(connectbus)
	go am.disconnectRoutine(disconnectbus)
	go am.presenceRoutine(presencebus)

	// Handle each connection.
	for {
		conn, err := listener.Accept()

		if err != nil {
			//l.Error(fmt.Sprintf("Could not accept connection: %s", err.Error()))
			log.Printf("Could not accept connection: %v\n", err.Error())
			os.Exit(1)
		}

		go xmppServer.TCPAnswer(conn)
	}
}
