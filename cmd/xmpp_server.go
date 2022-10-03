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
	//log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	envPort := 5222
	logLevel := logger.DebugLevel
	envSkipTLS := true
	envDomain := "localhost"
	envSelfXmppClient := selfXMppServerClient
	envSelfXmppClientPassword := selfXmppServerClientPassword

	// portPtr := flag.Int("port", envPort, "port number to listen on")
	// logLevelPtr := flag.Int("loggerLevel", logLevel, "set level logging")
	// flag.Parse()

	var adminUser = account.AdminUser{Name: envSelfXmppClient, Password: envSelfXmppClientPassword}

	//var registered = make(map[string]string)

	var activeUsers = make(map[string]chan<- interface{})

	var l = logger.Logger{Level: logger.Level(logLevel)}

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

	// l.Info("starting server")
	log.Println("[ms] starting server")
	// l.Info("listening on localhost:" + fmt.Sprintf("%d", *portPtr))
	log.Printf("[ms] listening on localhost: %v\n", envPort)

	// Listen for incoming connections.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", envPort))
	if err != nil {
		l.Error(fmt.Sprintf("[ms][error] could not listen for connections: %s", err.Error()))
		os.Exit(1)
	}
	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			log.Println("[ms][error]: ", closeErr)
		}
	}()

	// message Routines
	go am.RouteRoutine(messageBus)
	go am.ConnectRoutine(connectBus)
	go am.DisconnectRoutine(disconnectBus)
	go am.PresenceRoutine(presenceBus)

	// TODO: HeartBeatRouting()
	//go am.HeartBeatRoutine()

	// DEBUG TEST
	// xmppServer.Log.Info("=========== DEBUG TEST ===========")
	// xmppServer.Log.Error("Error LEVEL")
	// xmppServer.Log.Waring("Waring LEVEL")
	// xmppServer.Log.Info("Info LEVEL")
	// xmppServer.Log.Debug("Debug LEVEL")

	// Handle each connection.
	for {
		log.Println("[ms] start listen")
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
