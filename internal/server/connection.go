package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"
	"reflect"
)

// Connection represents a connection to an XMPP server.
type Connection struct {
	Raw          net.Conn
	MessageTypes map[xml.Name]reflect.Type
	out          *xml.Encoder
	in           *xml.Decoder
}

// NewConn creates a Connection struct for a given net.Conn and message system
func NewConn(raw net.Conn, MessageTypes map[xml.Name]reflect.Type) *Connection {
	log.Println("[c] NewConn: raw is net.Conn")
	conn := &Connection{
		Raw:          raw,
		MessageTypes: MessageTypes,
		in:           xml.NewDecoder(raw),
		out:          xml.NewEncoder(raw),
	}
	return conn
}

// Next scans the stream to find the next xml.StartElement
func (c *Connection) Next() (xml.StartElement, error) {
	// loop until a start element token is found
	for {
		nextToken, err := c.in.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		switch token := nextToken.(type) {
		case xml.StartElement:
			return token, nil
		default:
			log.Println("not found token")
		}
	}
}

// NextExt scans the stream to find the next xml.StartElement or xml.CharData
func (c *Connection) NextExt() (interface{}, error) {
	// loop until a start element token is found
	for {
		nextToken, err := c.in.Token()
		if err != nil {
			return xml.StartElement{}, err
		}
		switch token := nextToken.(type) {
		case xml.StartElement:
			return token, nil

		case xml.CharData:
			return token, nil
		}
	}
}

// Read the Element from the stream and reflect interface to known message types
func (c *Connection) Read(se xml.StartElement) (xml.Name, interface{}, error) {
	// Put start element in an interface and allocate one.
	var messageInterface interface{}

	if messageType, present := c.MessageTypes[se.Name]; present {
		messageInterface = reflect.New(messageType).Interface()
	} else {
		return xml.Name{}, nil, errors.New("Unknown XMPP message " + se.Name.Space + " <" + se.Name.Local + "/>")
	}

	// Unmarshal into that storage.
	if err := c.in.DecodeElement(messageInterface, &se); err != nil {
		return xml.Name{}, nil, err
	}

	return se.Name, messageInterface, nil
}

// SendStanza XML encodes the interface and sends it across the connection
func (c *Connection) SendStanza(s interface{}) error {
	data, err := xml.Marshal(s)
	if err != nil {
		log.Printf("[c] SendStanza Marshal err: %v\n", err.Error())
		return err
	}

	var n int
	n, err = c.Raw.Write(data)
	if err != nil {
		log.Printf("[c] SendStanza Write err: %v\n", err.Error())
	}
	log.Printf("[c] SendStanza Write data(%v): %v\n", n, data)
	return err
}

// SendRaw sends the string across the connection
func (c *Connection) SendRaw(s string) error {
	n, err := c.Raw.Write([]byte(s))
	if err != nil {
		log.Printf("[c] SendRaw Write err: %v\n", err.Error())
	}
	log.Printf("[c] SendRaw Write data(%v): %v\n", n, s)
	return err
}

// SendRawf formats and sends a string across the connection
func (c *Connection) SendRawf(format string, a ...interface{}) error {
	n, err := fmt.Fprintf(c.Raw, format, a...)
	if err != nil {
		log.Printf("[c] SendRawf Write err: %v\n", err.Error())
	}
	log.Printf("[c] SendRawf Write data(%v): ", n)
	log.Printf(format+"\n", a)
	return err
}
