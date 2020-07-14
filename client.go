package concord

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
)

const (
	SOM = 0x0a
	ACK = 0x06
	NAK = 0x15
)

type Client struct {
	port       string
	io         *serial.Port
	writeQueue chan []byte
	eventQueue chan Event

	ZoneMap   map[int]*Zone
	PanelType PanelType
	HWVersion string
	SWVersion string
	Serial    string
}

type EventType int

const (
	EventTypeZoneDefined  EventType = 0
	EventTypeZoneUpdated  EventType = 1
	EventTypePanelDefined EventType = 2
)

type Event struct {
	Type EventType
	Zone *Zone
}

func init() {
	tsf := new(log.TextFormatter)
	tsf.TimestampFormat = "2006-01-02 15:04:05.000"
	tsf.FullTimestamp = true
	log.SetFormatter(tsf)
}

func NewClient(port string) (*Client, error) {
	client := &Client{port: port}
	client.ZoneMap = make(map[int]*Zone)
	client.writeQueue = make(chan []byte, 10)
	client.eventQueue = make(chan Event, 10)

	// Seed write queue with default message - equipment discovery
	client.writeQueue <- msgEquipList

	if err := client.start(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) EventQueue() <-chan Event {
	return (<-chan Event)(c.eventQueue)
}

func (c *Client) start() error {
	io, err := serial.OpenPort(&serial.Config{Name: c.port, Baud: 9600, Parity: serial.ParityOdd, ReadTimeout: time.Millisecond * 125})
	if err != nil {
		return err
	}
	c.io = io

	// Startup read loop
	go c.ioLoop()

	return nil
}

func (c *Client) Close() error {
	return c.io.Close()
}
