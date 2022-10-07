package server

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// State processes the stream and moves to the next state
type State interface {
	Process(c *Connection, client *Client, s *Server) (State, *Connection, error)
}

// NewTLSStateMachine return steps through TCP TLS state
func NewTLSStateMachine(skipTLS bool) State {
	normal := &Normal{}
	if skipTLS {
		log.Println("[st] SkipTLS is true")
		authedStream := &AuthedStream{Next: normal}
		authedStart := &AuthedStart{Next: authedStream}
		auth := &Auth{Next: authedStart}
		start := &Start{Next: auth}
		return start
	} else {
		authedStream := &AuthedStream{Next: normal}
		authedStart := &AuthedStart{Next: authedStream}
		tlsAuth := &TLSAuth{Next: authedStart}
		tlsStartStream := &TLSStartStream{Next: tlsAuth}
		tlsUpgrade := &TLSUpgrade{Next: tlsStartStream}
		firstStream := &TLSUpgradeRequest{Next: tlsUpgrade}
		start := &Start{Next: firstStream}
		return start
	}
}

// Start state
type Start struct {
	Next State
}

// Process message
func (state *Start) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][Start] Start Process <<<<<")
	defer log.Printf("[st][Start] End Process >>>>>\n\n")

	log.Println("[st][Start] wait client next...")
	se, err := c.Next()
	if err != nil {
		return nil, c, err
	}

	log.Println("[st][Start] Read Start element:", se)

	// TODO: check that se is a stream
	sendErr := c.SendRawf("<?xml version='1.0'?><stream:stream id='%x' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>", createCookie())
	if sendErr != nil {
		log.Println("[st][Start][error]", sendErr)
	}

	if s.SkipTLS {
		log.Println("[st][Start] None TLS")
		// org
		//sendErr = c.SendRaw("<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>PLAIN</mechanism><mechanism>X-OAUTH2</mechanism></mechanisms></stream:features>")
		// support only PLAIN
		sendErr = c.SendRaw("<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>PLAIN</mechanism></mechanisms></stream:features>")
		if sendErr != nil {
			log.Println("[st][Start][error]: ", sendErr)
		}
	} else {
		log.Println("TLS")
		sendErr = c.SendRaw("<stream:features><starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'><required/></starttls></stream:features>")
		if sendErr != nil {
			log.Println("[st][Start][error]: ", sendErr)
		}
	}
	return state.Next, c, nil
}

// TLSUpgradeRequest state
type TLSUpgradeRequest struct {
	Next State
}

// Process message
func (state *TLSUpgradeRequest) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][TLSUpgradeRequest] Start Process <<<<<")
	defer log.Printf("[st][TLSUpgradeRequest] End Process >>>>>\n\n")

	log.Println("[st][TLSUpgradeRequest] wait client next...")
	_, err := c.Next()
	if err != nil {
		return nil, c, err
	}
	// TODO: ensure urn:ietf:params:xml:ns:xmpp-tls
	return state.Next, c, nil
}

// TLSUpgrade state
type TLSUpgrade struct {
	Next State
}

// Process message
func (state *TLSUpgrade) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][TLSUpgrade] Start Process <<<<<")
	defer log.Printf("[st][TLSUpgrade] End Process >>>>>\n\n")

	sendErr := c.SendRaw("<proceed xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>")
	if sendErr != nil {
		log.Println("[st][error]: ", sendErr)
	}

	// perform the TLS handshake
	tlsConn := tls.Server(c.Raw, s.TLSConfig)
	err := tlsConn.Handshake()
	if err != nil {
		return nil, c, err
	}
	// restart the Connection
	c = NewConn(tlsConn, c.MessageTypes)
	return state.Next, c, nil
}

// TLSStartStream state
type TLSStartStream struct {
	Next State
}

// Process messages
func (state *TLSStartStream) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][TLSStartStream] Start Process <<<<<")
	defer log.Printf("[st][TLSStartStream] End Process >>>>>\n\n")

	log.Println("[st][TLSStartStream] wait client next...")
	_, err := c.Next()
	if err != nil {
		return nil, c, err
	}
	// TODO: ensure check that se is a stream
	sendErr := c.SendRawf("<?xml version='1.0'?><stream:stream id='%x' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>", createCookie())
	if sendErr != nil {
		log.Println("[st][error]: ", sendErr)
	}
	sendErr = c.SendRaw("<stream:features><mechanisms xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><mechanism>PLAIN</mechanism></mechanisms></stream:features>")
	if sendErr != nil {
		log.Println("[st][error]: ", sendErr)
	}
	return state.Next, c, nil
}

// TLSAuth state
type TLSAuth struct {
	Next State
}

// Process messages
func (state *TLSAuth) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][TLSAuth] Start Process <<<<<")
	defer log.Printf("[st][TLSAuth] End Process >>>>>\n\n")

	log.Println("[st][TLSAuth] wait client next...")
	se, err := c.Next()
	if err != nil {
		return nil, c, err
	}
	// TODO: check what client sends, auth or register

	// read the full auth stanza
	_, val, err := c.Read(se)
	if err != nil {
		//s.Log.Error(errors.New("Unable to read auth stanza").Error())
		log.Println("[st][TLSAuth]Unable to read auth stanza")
		return nil, c, err
	}
	switch v := val.(type) {
	case *saslAuth:
		data, decodeErr := base64.StdEncoding.DecodeString(v.Body)
		if decodeErr != nil {
			return nil, c, decodeErr
		}
		info := strings.Split(string(data), "\x00")
		// should check that info[1] starts with client.jid
		success, authErr := s.Accounts.Authenticate(info[1], info[2])
		if authErr != nil {
			return nil, c, authErr
		}
		if success {
			client.localPart = info[1]
			sendErr := c.SendRaw("<success xmlns='urn:ietf:params:xml:ns:xmpp-sasl'/>")
			if sendErr != nil {
				log.Println("[st][TLSAuth][error]: ", sendErr)
			}

		} else {
			sendErr := c.SendRaw("<failure xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><not-authorized/></failure>")
			if sendErr != nil {
				log.Println("[st][TLSAuth][error]: ", sendErr)
			}
		}
	default:
		// expected authentication
		//s.Log.Error(errors.New("Expected authentication").Error())
		log.Println("[st][TLSAuth] Expected authentication")
		return nil, c, err
	}
	return state.Next, c, nil
}

type Auth struct {
	Next State
}

func (state *Auth) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][Auth] Start Process <<<<<")
	defer log.Printf("[st][Auth] End Process >>>>>\n\n")

	log.Println("[st][Auth] wait client next...")
	se, err := c.Next()
	if err != nil {
		log.Printf("[st][Auth] Auth (%v) Error : %v\n", client.jid, err)
		return nil, c, err
	}
	log.Printf("[st][Auth] Auth next full auth stanza:%v\n", se)

	//s.Log.Info(fmt.Sprintf("Auth read : %v", se))
	//log.Printf("[st][Auth] Auth read: %v\n", se)
	// TODO: check what client sends, auth or register

	// read the full auth stanza
	name, val, readErr := c.Read(se)
	if readErr != nil {
		//s.Log.Error(errors.New("Unable to read auth stanza").Error())
		log.Println("[st][Auth] Unable to read auth stanza")
		return nil, c, readErr
	}

	//s.Log.Info(fmt.Sprintf("Auth read full auth stanza : %v", se))
	log.Printf("[st][Auth] Auth read name: %v, val: %v \n", name, val)

	switch v := val.(type) {
	case *saslAuth:
		data, decodeErr := base64.StdEncoding.DecodeString(v.Body)
		if decodeErr != nil {
			return nil, c, decodeErr
		}
		info := strings.Split(string(data), "\x00")
		// should check that info[1] starts with client.jid
		success, authErr := s.Accounts.Authenticate(info[1], info[2])
		if authErr != nil {
			return nil, c, authErr
		}

		if success {
			client.localPart = info[1]
			sendErr := c.SendRaw("<success xmlns='urn:ietf:params:xml:ns:xmpp-sasl'/>")
			if sendErr != nil {
				log.Println("[st][Auth][error]: ", sendErr)
			}
		} else {
			sendErr := c.SendRaw("<failure xmlns='urn:ietf:params:xml:ns:xmpp-sasl'><not-authorized/></failure>")
			if sendErr != nil {
				log.Println("[st][Auth][error]: ", sendErr)
			}
			return nil, c, errors.New("client not authorized")
		}
	default:
		// expected authentication
		//s.Log.Error(errors.New("Expected authentication").Error())
		log.Println("[st][Auth] Expected authentication")
		return nil, c, err
	}
	return state.Next, c, nil
}

// AuthedStart state
type AuthedStart struct {
	Next State
}

// Process messages
func (state *AuthedStart) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][AuthedStart] Start Process <<<<<")
	defer log.Printf("[st][AuthedStart] End Process >>>>>\n\n")

	log.Println("[st][AuthedStart] wait client next...")
	se, err := c.Next()
	if err != nil {
		return nil, c, err
	}
	log.Println("[st][AuthedStart] Read Start element:", se)

	sendErr := c.SendRawf("<?xml version='1.0'?><stream:stream id='%x' version='1.0' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams'>", createCookie())
	if sendErr != nil {
		log.Println("[st][AuthedStart][error]: ", sendErr)
	}

	// org
	//sendErr = c.SendRaw("<stream:features><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/></stream:features>")

	// old
	// sendErr = c.SendRaw("<stream:features><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>"+
	// "<session xmlns='urn:ietf:params:xml:ns:xmpp-session'><optional/></session>"+
	// "<c ver='LcF33OEjnzEcDbJUF4hNy/ifCdE=' node='http://auth.test.com/' hash='sha-1' xmlns='http://jabber.org/protocol/caps'/>"+
	// "<ver xmlns='urn:xmpp:features:rosterver'/>"+
	// "<keepalive xmlns='urn:xmpp:keepalive:0'><interval min='60' max='300'/></keepalive></stream:features>")

	sendErr = c.SendRaw("<stream:features><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'/>" +
		"<keepalive xmlns='urn:xmpp:keepalive:0'><interval min='30' max='300'/></keepalive></stream:features>")
	if sendErr != nil {
		log.Println("[st][AuthedStart][error]: ", sendErr)
	}

	return state.Next, c, nil
}

// AuthedStream state
type AuthedStream struct {
	Next State
}

// Process messages
func (state *AuthedStream) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][AuthedStream] Start Process <<<<<")
	defer log.Printf("[st][AuthedStream] End Process >>>>>\n\n")

	log.Println("[st][AuthedStream] wait client next...")
	se, err := c.Next()
	if err != nil {
		return nil, c, err
	}

	log.Println("[st][AuthedStream] Read Start element:", se)

	// check that it's a bind request
	// read bind request
	name, val, err := c.Read(se)
	if err != nil {
		return nil, c, err
	}
	log.Println("[st][AuthedStream] Read Start name:", name)
	log.Println("[st][AuthedStream] Read val:", val)

	switch v := val.(type) {
	case *ClientIQ:
		// TODO: actually validate that it's a bind request
		if v.Bind.Resource == "" {
			client.resourcePart = makeResource()
		} else {
			client.resourcePart = v.Bind.Resource
			client.domainPart = s.Domain
			//s.Log.Error(errors.New("Invalid bind request").Error())
			//return nil, c, err
		}

		log.Println("[st][AuthedStream] ID: ", v.ID)
		log.Println("[st][AuthedStream] To: ", v.To)
		log.Println("[st][AuthedStream] From: ", v.From)
		log.Println("[st][AuthedStream] Query: ", v.Query)

		log.Println("[st][AuthedStream] Client localPart(Username)", client.localPart)
		log.Println("[st][AuthedStream] Client DomainPart", client.domainPart)
		log.Println("[st][AuthedStream] Client ResourcePart", client.resourcePart)

		client.jid = client.localPart + "@" + client.domainPart + "/" + client.resourcePart
		sendErr := c.SendRawf("<iq id='%s' type='result'><bind xmlns='urn:ietf:params:xml:ns:xmpp-bind'><jid>%s</jid></bind></iq>", v.ID, client.jid)
		if sendErr != nil {
			log.Println("[st][AuthedStream][error]: ", sendErr)
		}
		s.ConnectBus <- Connect{Jid: client.jid,LocalPart: client.localPart, Receiver: client.messages}

		// create and set client's socket deadline
		if s.DeadlineEnable {
			setDeadline(s.DeadlineInterval, s.DeadlineMaxCount, &c.Raw, client.jid)
		}

	default:
		//s.Log.Error(errors.New("Expected ClientIQ message").Error())
		log.Println("[st][AuthedStream]Expected ClientIQ message")
		return nil, c, err
	}
	return state.Next, c, nil
}

// Normal state
type Normal struct{}

// Process messages
func (state *Normal) Process(c *Connection, client *Client, s *Server) (State, *Connection, error) {
	log.Println("[st][Normal] Start Process <<<<<")
	defer log.Printf("[st][Normal] End Process >>>>>\n\n")
	var err error
	nextReadDone := make(chan bool)
	nextReadErr := make(chan error)

	// one go routine to read/respond
	go func(done chan bool, errors chan error) {
		for {
			log.Println("[st][Normal] wait client next...")
			ex, nextErr := c.NextExt()
			if nextErr != nil {
				log.Printf("[st][Normal][error] err: %v\n", nextErr.Error())
				errors <- nextErr
				done <- true
				return
			}

			switch data := ex.(type) {
			case xml.StartElement:
				log.Printf("[st][Normal] start element: %v\n", data)
				name, val, readErr := c.Read(data)
				if readErr != nil {
					log.Printf("[st][Normal] Read Error: %v\n", err.Error())
				} else {
					log.Printf("[st][Normal] Read Name[%v]: %v\n", name, val)
				}

				if parsed, ok := val.(*ClientIQ); ok {
					// Receive ping from client. *reference XEP-0199: XMPP Ping
					if parsed.Type == "get" && parsed.ID == "c2s1" {
						log.Println("[ex] receive ping")
						// response pong message
						msg := fmt.Sprintf("<iq from='%v' to='%v' id='c2s1' type='result'/>", parsed.To, parsed.From)
						client.messages <- msg

						if s.DeadlineEnable {
							// set Deadline
							setDeadline(s.DeadlineInterval, s.DeadlineMaxCount, &c.Raw, client.jid)
						}
					}
				}

				for _, extension := range s.Extensions {
					extension.Process(val, client)
				}

			case xml.CharData:
				// https://xmpp.org/rfcs/rfc6120.html#xml-whitespace
				// rfc6120, section 1.4: "whitespace" is used to refer to any character
				// or characters matching [...] SP, HTAB, CR, or LF.
				switch string(data) {
				case " ", "\t", "\r", "\n": //TODO: consider more than one whitespace
					log.Println("[st][Normal] received whitespace ping")

					if s.DeadlineEnable {
						// set Deadline
						setDeadline(s.DeadlineInterval, s.DeadlineMaxCount, &c.Raw, client.jid)
					}
				}
			}
		}
	}(nextReadDone, nextReadErr)

	for {
		select {
		case messages := <-client.messages:
			log.Println("[st][Normal] client messages")
			switch msg := messages.(type) {
			default:
				log.Println("[st][Normal] default type")
				err = c.SendStanza(msg)

			case string:
				log.Println("[st][Normal] string type")
				err = c.SendRaw(msg)

			}
			if err != nil {
				nextReadErr <- err
			}

		case <-nextReadDone:
			log.Println("[st][Normal] <- nextReadDone")
			return nil, c, nil

		case connErr := <-nextReadErr:
			//s.Log.Error(fmt.Sprintf("Connection Error: %s", err.Error()))
			log.Printf("[st][Normal] nextRead Error: %v\n", connErr.Error())
		}
	}
}

// setDeadline set client's socket deadline
func setDeadline(sec, count int, conn *net.Conn, jid string) {
	addTime := time.Second * time.Duration(sec * count)
	timeDeadline := time.Now().Add(addTime)
	setDeadlineErr := (*conn).SetReadDeadline(timeDeadline)
	if setDeadlineErr != nil {
		log.Println("[st][hb] err: ", setDeadlineErr)
	} else {
		log.Printf("[st][hb] %v(%v) setReadDeadline: %v", jid, addTime, timeDeadline)
	}
}