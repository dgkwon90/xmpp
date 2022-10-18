package main

import (
	"log"
	"runtime"
	"xmpp/internal/account"
	"xmpp/internal/pkg/amqp/rabbitmq"
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

	/*
	 * define env value
	 */
	envPort := 5222
	envLogLevel := logger.DebugLevel
	envDomain := "localhost"

	// TLS
	envSkipTLS := true
	envTLSCertFile := "F:/workspace/cert/key/server.pem"
	envTLSKeyFile := "F:/workspace/cert/key/server.key"

	// Deadline
	envDeadlineEnable := false
	envDeadlineInterval := 30
	envDeadlineMaxCount := 3

	// Account Info
	envAccountManagementID := "xmpp-server.test" //+ generate.UUID()

	// Self Password
	envSkipPassword := true
	envSelfXmppClient := "selfXmppClient"
	envSelfXmppClientPassword := "test1234"
	envSelfXMppClientResource := "XMPPConn1"

	// DB Redis
	envUseDB := true
	envDBType := "redis"
	envRedisConfHost := "192.168.56.1"
	envRedisConfPort := 6379
	envRedisConfPassword := "test001"
	envRedisConfPool := 10000
	envRedisConfTimeout := 0
	envRedisConfClusters := ""

	// MQ Rabbitmq (publisher)
	envUseMQ := true
	envMQType := "rabbitmq"
	envRabbitmqUrl := "amqp://dgkwon:test001@192.168.56.1:5672/"
	envRabbitmqPublisherName := "xmpp-server-publisher"

	envRabbitmqPublisherTopic := "device-connection-topic"
	envRabbitmqPublisherQueueName := "device-connection-topic-pull"

	// MQ Message (Consumer/Subscribe)
	envRabbitmqConsumerName := envAccountManagementID + "-consumer"
	envRabbitmqConsumeExchange := envAccountManagementID
	envRabbitmqConsumerQueueName := envAccountManagementID + "-pull"

	// l.Info("listening on localhost:" + fmt.Sprintf("%d", *portPtr))
	log.Println("======== env values ========")
	log.Printf("[ms] listening on localhost: %v\n", envPort)
	log.Printf("[ms] debug level: %v\n", envLogLevel)
	log.Printf("[ms] domain: %v\n", envDomain)

	log.Printf("[ms] skip tls: %v\n", envSkipTLS)
	log.Printf("[ms] tls keyfile: %v\n", envTLSKeyFile)
	log.Printf("[ms] tls certfile: %v\n", envTLSCertFile)

	log.Printf("[ms] deadline enable: %v\n", envDeadlineEnable)
	log.Printf("[ms] deadline Interval: %v\n", envDeadlineInterval)
	log.Printf("[ms] deadline MaxcCount: %v\n", envDeadlineMaxCount)

	log.Printf("[ms] accounnt management id: %v\n", envAccountManagementID)
	log.Printf("[ms] Skip Password: %v\n", envSkipPassword)
	log.Printf("[ms] admin client: %v\n", envSelfXmppClient)
	log.Printf("[ms] admin password: %v\n", envSelfXmppClientPassword)
	log.Printf("[ms] admin resource: %v\n", envSelfXMppClientResource)

	log.Printf("[ms] ues db: %v\n", envUseDB)
	log.Printf("[ms] db type: %v\n", envDBType)
	log.Printf("[ms] redis conf host: %v\n", envRedisConfHost)
	log.Printf("[ms] redis conf port: %v\n", envRedisConfPort)
	log.Printf("[ms] redis conf password: %v\n", envRedisConfPassword)
	log.Printf("[ms] redis conf pool: %v\n", envRedisConfPool)
	log.Printf("[ms] redis conf timeout: %v\n", envRedisConfTimeout)
	log.Printf("[ms] redis conf cluster: %v\n", envRedisConfClusters)

	log.Printf("[ms] ues mq: %v\n", envUseMQ)
	log.Printf("[ms] nq type: %v\n", envMQType)
	log.Printf("[ms] rabbitmq url: %v\n", envRabbitmqUrl)
	log.Printf("[ms] rabbitmq publisher name: %v\n", envRabbitmqPublisherName)
	log.Printf("[ms] rabbitmq publisher topic: %v\n", envRabbitmqPublisherTopic)
	log.Printf("[ms] rabbitmq publisher queuename: %v\n", envRabbitmqPublisherQueueName)

	log.Printf("[ms] rabbitmq consumer name: %v\n", envRabbitmqConsumerName)
	log.Printf("[ms] rabbitmq consume exchange(topic): %v\n", envRabbitmqConsumeExchange)
	log.Printf("[ms] rabbitmq consumer queuename: %v\n", envRabbitmqConsumerQueueName)

	log.Println("==============================")
	// portPtr := flag.Int("port", envPort, "port number to listen on")
	// logLevelPtr := flag.Int("loggerLevel", envLogLevel, "set level logging")
	// flag.Parse()

	var adminUser = account.AdminUser{
		Name:     envSelfXmppClient,
		Domain:   envDomain,
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
	var connectionRequestBus = make(chan server.ConnectionRequest)

	var am = account.Management{
		AdminUser:    adminUser,
		SkipPassword: envSkipPassword,
		/*Users: registered,*/
		Online:            activeUsers,
		Log:               l,
		Mutex:             &sync.Mutex{},
		UseDB:             envUseDB,
		DBType:            envDBType,
		UseMQ:             envUseMQ,
		MQType:            envMQType,
		ID:                envAccountManagementID,
		ConnectionRequest: make(map[string]chan bool),
	}

	log.Printf("[ms] UseDB: %v\n", am.UseDB)
	if am.UseDB {
		log.Printf("[ms] setup DBType: %v\n", am.DBType)
		if am.DBType == "redis" {
			log.Println("[ms] account management use redis enable")
			conf := &memstore.Config{
				Timeout:  envRedisConfTimeout,
				Pool:     envRedisConfPool,
				Password: envRedisConfPassword,
				Clusters: envRedisConfClusters,
			}
			conf.Address.Host = envRedisConfHost
			conf.Address.Port = envRedisConfPort
			am.DB = memstore.NewClient(conf)
		}
	}

	log.Printf("[ms] UseMQ: %v\n", am.UseMQ)
	if am.UseMQ {
		log.Printf("[ms] setup MQType: %v\n", am.MQType)
		if am.MQType == "rabbitmq" {
			log.Println("[ms] setup publisher")
			am.Publisher = rabbitmq.NewClient(
				envRabbitmqPublisherName,
				"",
				envRabbitmqUrl,
			)

			log.Println("[ms] setup consume routine")
			go am.ConsumeRoutine(
				envRabbitmqConsumeExchange,
				envRabbitmqConsumerName,
				envRabbitmqConsumerQueueName,
				envRabbitmqUrl,
			)
		}
	}

	var cert, certErr = tls.LoadX509KeyPair(envTLSCertFile, envTLSKeyFile)
	if certErr != nil && !envSkipTLS {
		log.Printf("[ms] LoadX509KeyPair error %v", certErr)
		os.Exit(1)
	}

	var tlsConfig = tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		CipherSuites: []uint16{
			// ADD
			// tls.TLS_RSA_WITH_RC4_128_SHA,
			// tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			// tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			// tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			// tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
			// tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			// tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,

			// tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			// tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

			// ORG
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
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

		// Connection request
		ConnectionRequestBus: connectionRequestBus,
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
	go am.ConnectRoutine(connectBus)
	go am.DisconnectRoutine(disconnectBus)

	go am.RouteRoutine(messageBus)

	go am.PresenceRoutine(presenceBus)

	// add connection request message Routine
	go am.ConnectionRequestRoutine(connectionRequestBus)

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
