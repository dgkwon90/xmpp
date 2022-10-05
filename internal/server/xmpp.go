// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xmpp implements the XMPP IM protocol, as specified in RFC 6120 and
// 6121.
package server

import (
	"crypto/tls"
	"log"
	"net"
	"xmpp/internal/keepalive"
)

// Client xmpp connection
type Client struct {
	jid          string
	localPart    string
	domainPart   string
	resourcePart string
	messages     chan interface{}
	heartbeat    *keepalive.Heartbeat
}

// AccountManager performs roster management and authentication
type AccountManager interface {
	Authenticate(username, password string) (success bool, err error)
	//CreateAccount(username, password string) (success bool, err error)
	OnlineRoster(jid string) (online []string, err error)
}

// Logging interface for library messages
type Logging interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Waring(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// Server contains options for an XMPP connection.
type Server struct {
	// what domain to use?
	Domain string

	// SkipTLS, if true, causes the TLS handshake to be skipped.
	// WARNING: this should only be used if Conn is already secure.
	SkipTLS bool
	// TLSConfig contains the configuration to be used by the TLS
	// handshake. If nil, sensible defaults will be used.
	TLSConfig *tls.Config

	// AccountManager handles messages that the server must respond to
	// such as authentication and roster management
	Accounts AccountManager

	// Extensions are injectable handlers that process messages
	Extensions []Extension

	// How the client notifies the server who the connection is
	// and how to send messages to the connection JID
	ConnectBus chan<- Connect

	// notify server that the client has disconnected
	DisconnectBus chan<- Disconnect

	// Injectable logging interface
	Log Logging
}

// Message is a generic XMPP message to send to the To Jid
type Message struct {
	To   string
	Data interface{}
}

// Connect holds a channel where the server can send messages to the specific Jid
type Connect struct {
	Jid      string
	Receiver chan<- interface{}
}

// Disconnect notifies when a jid disconnects
type Disconnect struct {
	Jid string
}

// TCPAnswer sends connection through the TSLStateMachine
func (s *Server) TCPAnswer(conn net.Conn) {
	log.Printf("\n\n")
	log.Println("[x] client connection")
	defer func() {
		closeErr := conn.Close()
		if closeErr != nil {
			log.Println("[x][ERROR] network connection close : ", closeErr)
		}
	}()

	log.Printf("[x] Accepting TCP connection from: %v\n", conn.RemoteAddr())
	log.Printf("[x] connection LocalAddr: %v\n", conn.LocalAddr())

	state := NewTLSStateMachine(s.SkipTLS)

	// create new client
	client := &Client{
		messages: make(chan interface{}),
	}
	defer close(client.messages)

	clientConnection := NewConn(conn, MessageTypes)
	for {
		var err error
		state, clientConnection, err = state.Process(clientConnection, client, s)
		//s.Log.Debug(fmt.Sprintf("[state] %s", state))

		if err != nil {
			//s.Log.Error(fmt.Sprintf("[%s] State Error: %s", client.jid, err.Error()))
			log.Printf("[x][%v] State Error: %v\n", client.jid, err.Error())
			return
		}
		if state == nil {
			//s.Log.Info(fmt.Sprintf("Client Disconnected: %s", client.jid))
			log.Printf("[x] Client Disconnected:  %v\n", client.jid)
			s.DisconnectBus <- Disconnect{Jid: client.jid}
			return
		}
	}
}
