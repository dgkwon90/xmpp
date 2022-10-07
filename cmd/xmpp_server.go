package main

import (
	"log"
	"runtime"
	"xmpp/internal/account"
	"xmpp/internal/pkg/logger"
	"xmpp/internal/pkg/memstore"
	"xmpp/internal/server"

	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
)

/* Main server loop */
func main() {
	// set flag for log
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// l.Info("starting server")
	log.Println("[ms] starting server")

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("[ms] CPU PROCS: ", runtime.GOMAXPROCS(0))

	// define env value
	envPort := 5222
	envLogLevel := logger.DebugLevel
	envSkipTLS := true
	envDomain := "localhost"

	envDeadlineEnable := false
	envDeadlineInterval := 30
	envDeadlineMaxCount := 3

	envSelfXmppClient := "selfXmppClient"
	envSelfXmppClientPassword := "test1234"
	envSelfXMppClientResource := "XMPPConn1"

	 envSkipPassword:= true

	envUseRedis := true
	envRedisConfHost := "localhost"
	envRedisConfPort := 6379
	envRedisConfPassword := ""
	envRedisConfPool := 10000
	envRedisConfTimeout := 0
	envRedisConfClusters := ""
	// Redis


	// l.Info("listening on localhost:" + fmt.Sprintf("%d", *portPtr))
	log.Println("======== env values ========")

	log.Printf("[ms] listening on localhost: %v\n", envPort)
	log.Printf("[ms] debug level: %v\n", envLogLevel)
	log.Printf("[ms] skip tls: %v\n", envSkipTLS)
	log.Printf("[ms] domain: %v\n", envDomain)

	log.Printf("[ms] deadline enable: %v\n", envDeadlineEnable)
	log.Printf("[ms] deadline Interval: %v\n", envDeadlineInterval)
	log.Printf("[ms] deadline MaxcCount: %v\n", envDeadlineMaxCount)

	log.Printf("[ms] admin client: %v\n", envSelfXmppClient)
	log.Printf("[ms] admin password: %v\n", envSelfXmppClientPassword)
	log.Printf("[ms] admin resource: %v\n", envSelfXMppClientResource)
	log.Printf("[ms] Skip Password: %v\n", envSkipPassword)

	log.Printf("[ms] ues redis: %v\n", envUseRedis)
	log.Printf("[ms] redis conf host: %v\n", envRedisConfHost)
	log.Printf("[ms] redis conf port: %v\n", envRedisConfPort)
	log.Printf("[ms] redis conf password: %v\n", envRedisConfPassword)
	log.Printf("[ms] redis conf pool: %v\n", envRedisConfPool)
	log.Printf("[ms] redis conf timeout: %v\n", envRedisConfTimeout)
	log.Printf("[ms] redis conf cluster: %v\n", envRedisConfClusters)

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

	var am = account.Management{AdminUser: adminUser, SkipPassword: envSkipPassword/*Users: registered,*/, Online: activeUsers, Log: l, Mutex: &sync.Mutex{}, UseRedis: envUseRedis}
	if am.UseRedis {
		log.Println("[ms] account management use redis enable")
		conf := &memstore.Config{
			Timeout:  envRedisConfTimeout,
			Pool:     envRedisConfPool,
			Password: envRedisConfPassword,
			Clusters: envRedisConfClusters,
		}
		conf.Address.Host = envRedisConfHost
		conf.Address.Port = envRedisConfPort
		am.Redis = memstore.NewClient(conf)
	}

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

		// Deadline option
		DeadlineEnable:   envDeadlineEnable,
		DeadlineInterval: envDeadlineInterval,
		DeadlineMaxCount: envDeadlineMaxCount,
	}

	// Listen for incoming connections.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", envPort))
	if err != nil {
		//l.Error(fmt.Sprintf("[ms][error] could not listen for connections: %s", err.Error()))
		log.Printf("[ms][error] could not listen for connections: %v", err)
		os.Exit(1)
	}
	// defer close listen struct
	defer func() {
		closeErr := listener.Close()
		if closeErr != nil {
			log.Println("[ms][error] close listener : ", closeErr)
		} else {
			log.Println("[ms] close listener")
		}
	}()

	// message Routines
	go am.RouteRoutine(messageBus)
	go am.ConnectRoutine(connectBus)
	go am.DisconnectRoutine(disconnectBus)
	go am.PresenceRoutine(presenceBus)

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
