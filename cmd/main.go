package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fmt"

	"strings"

	"github.com/cicavey/concord"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var discoverBase *string

func main() {

	mqttAddr := flag.String("mqtt", "", "MQTT server, host:port")
	usbPath := flag.String("device", "/dev/ttyUSB0", "USB device, fully qualified")
	statusTopic := flag.String("statusTopic", "hass/status", "Home Assistant Birth/Will Topic")
	discoverBase = flag.String("discoverBase", "homeassistant", "Home Assistant base discovery topic")

	flag.Parse()

	// MQTT
	opts := MQTT.NewClientOptions()
	opts.AddBroker(*mqttAddr)
	opts.SetClientID("concord")
	opts.SetCleanSession(true)
	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to hass status topic so we can rebroacast when hass comes up/down
	status := make(chan [2]string)
	if token := client.Subscribe(*statusTopic, 0, func(client MQTT.Client, msg MQTT.Message) {
		status <- [2]string{msg.Topic(), string(msg.Payload())}
	}); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Serial
	c, err := concord.NewClient(*usbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	done := setupSigHandler()

	ticker := time.NewTicker(time.Second * 30)

loopbreak:
	for {
		select {
		case evt := <-c.EventQueue():
			if evt.Type == 0 {
				publishZone(evt.Zone, client)
			} else if evt.Type == 1 {
				updateZone(evt.Zone, client)
			}
		case msg := <-status:
			fmt.Println(msg)
			if msg[1] == "online" {
				// Rebroadcast Zones
				for _, z := range c.ZoneMap {
					publishZone(z, client)
				}
			}
		case <-ticker.C:
			fmt.Println("tick")
			for _, z := range c.ZoneMap {
				publishZone(z, client)
			}
		case <-done:
			break loopbreak
		}
	}

	ticker.Stop()

	if token := client.Unsubscribe(*statusTopic); token.Wait() && token.Error() != nil {
		print(token.Error())
	}

	client.Disconnect(250)
}

func publishZone(z *concord.Zone, client MQTT.Client) {
	/* created
	homeassistant/binary_sensor/<ID>/config
	{"name": "garden", "device_class": "motion"}

	device_class:
	- motion (ON/OFF)
	- opening (ON OPEN, OFF CLOSED)

	homeassistant/binary_sensor/<ID>/state
	ON/OFF
	*/

	cid := fmt.Sprintf("concord_zone_%d", z.ID)

	lowerName := strings.ToLower(z.Name)

	// Guess at device class based on name - default opening
	deviceClass := "opening"
	if strings.Contains(lowerName, "motion") {
		deviceClass = "motion"
	}

	configTopic := *discoverBase + "/binary_sensor/" + cid + "/config"
	configValue := "{\"name\": \"" + strings.Title(lowerName) + "\", \"device_class\": \"" + deviceClass + "\"}"

	//client.Publish(configTopic, 0, true, "")
	client.Publish(configTopic, 0, true, configValue)

	stateTopic := *discoverBase + "/binary_sensor/" + cid + "/state"
	stateValue := "OFF"
	if z.Status != 0 {
		stateValue = "ON"
	}
	client.Publish(stateTopic, 0, true, stateValue)
}

func updateZone(z *concord.Zone, client MQTT.Client) {
	cid := fmt.Sprintf("concord_zone_%d", z.ID)
	stateTopic := *discoverBase + "/binary_sensor/" + cid + "/state"
	stateValue := "OFF"
	if z.Status != 0 {
		stateValue = "ON"
	}
	client.Publish(stateTopic, 0, true, stateValue)
}

func setupSigHandler() chan bool {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("Ctrl-C")
		done <- true
	}()
	return done
}
