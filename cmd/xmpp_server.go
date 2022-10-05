package main

import (
	"log"
	"xmpp/internal/account"
	"xmpp/internal/pkg/logger"
	"xmpp/internal/server"

	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
)

const (
	selfXMppServerClient         = "selfXmppClient"
	selfXmppServerClientPassword = "test1234"
	selfXmppServerClientResource = "XMPPConn1"
)

/* Main server loop */
func main() {
	// set flag for log
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// l.Info("starting server")
	log.Println("[ms] starting server")

	// define env value
	envPort := 5222
	envLogLevel := logger.DebugLevel
	envSkipTLS := true
	envDomain := "localhost"
	envSelfXmppClient := selfXMppServerClient
	envSelfXmppClientPassword := selfXmppServerClientPassword
	envSelfXMppClientResource := selfXmppServerClientResource

	// l.Info("listening on localhost:" + fmt.Sprintf("%d", *portPtr))
	log.Println("====== env values ======")
	log.Printf("[ms] listening on localhost: %v\n", envPort)
	log.Printf("[ms] debug level: %v\n", envLogLevel)
	log.Printf("[ms] skip tls: %v\n", envSkipTLS)
	log.Printf("[ms] domain: %v\n", envDomain)
	log.Printf("[ms] admin client: %v\n", envSelfXmppClient)
	log.Printf("[ms] admin password: %v\n", envSelfXmppClientPassword)
	log.Printf("[ms] admin resource: %v\n", envSelfXMppClientResource)
	log.Println("==============================")
	// portPtr := flag.Int("port", envPort, "port number to listen on")
	// logLevelPtr := flag.Int("loggerLevel", envLogLevel, "set level logging")
	// flag.Parse()

	var adminUser = account.AdminUser{
		Name:     envSelfXmppClient,
		Password: envSelfXmppClientPassword,
		Resource: envSelfXMppClientResource,
	}

	//var registered = make(map[string]string)

	var activeUsers = make(map[string]chan<- interface{})

	var l = logger.Logger{
		Level: logger.Level(envLogLevel),
	}

	var messageBus = make(chan server.Message)
	var presenceBus = make(chan server.Message)
	var connectBus = make(chan server.Connect)
	var disconnectBus = make(chan server.Disconnect)

	var am = account.Management{AdminUser: adminUser /*Users: registered,*/, Online: activeUsers, Log: l, Mutex: &sync.Mutex{}}

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

	xmppServer := &server.Server{
		SkipTLS:    envSkipTLS,
		Log:        l,
		Accounts:   am,
		ConnectBus: connectBus,
		Extensions: []server.Extension{
			&server.DebugExtension{Log: l},
			&server.NormalMessageExtension{MessageBus: messageBus},
			&server.RosterExtension{Accounts: am},
			&server.PresenceExtension{PresenceBus: presenceBus},
		},
		DisconnectBus: disconnectBus,
		Domain:        envDomain,
		TLSConfig:     &tlsConfig,
	}

	// Listen for incoming connections.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", envPort))
	if err != nil {
		l.Error(fmt.Sprintf("[ms][error] could not listen for connections: %s", err.Error()))
		os.Exit(1)
	}
	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			log.Println("[ms][error] listener close: ", closeErr)
		}
	}()

	// message Routines
	go am.RouteRoutine(messageBus)
	go am.ConnectRoutine(connectBus)
	go am.DisconnectRoutine(disconnectBus)
	go am.PresenceRoutine(presenceBus)
	//go am.KeepAliveRoutine(keepAliveBus)

	// Handle each connection.
	log.Println("[ms] start client connection routine...")
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			//l.Error(fmt.Sprintf("Could not accept connection: %s", err.Error()))
			log.Printf("[ms][error] could not accept connection: %v\n", err.Error())
			os.Exit(1)
		}
		//tcp go routine
		go xmppServer.TCPAnswer(conn)
	}
}
