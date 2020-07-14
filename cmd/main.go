package main

import (
	"crypto/tls"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fmt"

	"encoding/json"

	"strings"

	"github.com/cicavey/concord"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

var BuildVersion string = "standalone"
var BuildDate string = "now"

var config *viper.Viper

func main() {

	version := pflag.BoolP("version", "v", false, "Show version")
	pflag.Parse()

	if *version {
		fmt.Println(BuildVersion, BuildDate)
		os.Exit(0)
	}

	localConfig, err := ResolveConfig()
	if err != nil {
		log.Fatal(err)
	}
	config = localConfig

	if level, err := log.ParseLevel(config.GetString("log_level")); err == nil {
		// Print directly so that log level changes are always visible
		fmt.Println("Setting log level to", level)
		log.SetLevel(level)
	}

	// MQTT
	opts := MQTT.NewClientOptions()
	opts.AddBroker(config.GetString("mqtt.broker"))
	opts.SetClientID(config.GetString("mqtt.client_id"))
	opts.SetCleanSession(true)

	mqttUser := config.GetString("mqtt.username")
	if mqttUser != "" {
		opts.SetUsername(mqttUser)
		mqttPass := config.GetString("mqtt.password")
		if mqttPass != "" {
			opts.SetPassword(mqttPass)
		}
	}

	// TODO: Add proper support for root CAs, client certs. Better than nothing
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Subscribe to hass status topic so we can rebroacast when hass comes up/down
	status := make(chan [2]string)
	if token := client.Subscribe(config.GetString("homeassistant.status_topic"), 0, func(client MQTT.Client, msg MQTT.Message) {
		status <- [2]string{msg.Topic(), string(msg.Payload())}
	}); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Serial
	c, err := concord.NewClient(config.GetString("device"))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	done := setupSigHandler()

	ticker := time.NewTicker(time.Second * 30)

	publishAll := func() {
		for _, z := range c.ZoneMap {
			publishZone(c, z, client)
		}
	}

loopbreak:
	for {
		select {
		case evt := <-c.EventQueue():
			switch evt.Type {
			case concord.EventTypeZoneDefined:
				publishZone(c, evt.Zone, client)
			case concord.EventTypeZoneUpdated:
				updateZone(evt.Zone, client)
			case concord.EventTypePanelDefined:
				publishAll()
			}
		case msg := <-status:
			log.Info(msg)
			if msg[1] == "online" {
				// Rebroadcast Zones
				publishAll()
			}
		case <-ticker.C:
			log.Info("tick, republish zones")
			publishAll()
		case <-done:
			break loopbreak
		}
	}

	ticker.Stop()

	if token := client.Unsubscribe(config.GetString("homeassistant.status_topic")); token.Wait() && token.Error() != nil {
		log.Warn(token.Error())
	}

	client.Disconnect(250)
}

func stateTopic(z concord.Zone) string {
	cid := fmt.Sprintf("concord_zone_%d", z.ID)
	return config.GetString(ConfDiscoverBase) + "/binary_sensor/" + cid + "/state"
}

func stateValue(z concord.Zone) string {
	if z.Status != 0 {
		return "ON"
	}
	return "OFF"
}

type hadevice struct {
	Identifiers  string `json:"identifiers"`
	Manufacturer string `json:"manufacturer"` // Interlogix
	Model        string `json:"model"`        // hw_rev
	Name         string `json:"name"`         // panel type
	Version      string `json:"sw_version"`   // sw_ver
}

// This conforms to homeassistant's mqtt discovery definitions
type haentity struct {
	Name        string   `json:"name"`
	DeviceClass string   `json:"device_class"`
	StateTopic  string   `json:"state_topic"`
	UniqueID    string   `json:"unique_id"`
	Device      hadevice `json:"device"`
}

func publishZone(c *concord.Client, z *concord.Zone, client MQTT.Client) {

	// Only bother publishing if the
	if c.Serial == "" || c.SWVersion == "" || c.HWVersion == "" {
		log.Warn("Attempt to publish zones without panel definition")
		return
	}

	cid := fmt.Sprintf("concord_zone_%d", z.ID)

	lowerName := strings.ToLower(z.Name)

	// Guess at device class based on name - default opening
	deviceClass := "opening"
	if strings.Contains(lowerName, "motion") {
		deviceClass = "motion"
	}
	if strings.Contains(lowerName, "door") {
		deviceClass = "door"
	}
	if strings.Contains(lowerName, "window") {
		deviceClass = "window"
	}

	configTopic := config.GetString(ConfDiscoverBase) + "/binary_sensor/" + cid + "/config"

	configValue := haentity{
		Name:        strings.Title(lowerName),
		DeviceClass: deviceClass,
		StateTopic:  stateTopic(*z),
		UniqueID:    c.Serial + "-" + cid,
		Device: hadevice{
			Identifiers:  c.Serial,
			Manufacturer: "Interlogix",
			Model:        c.HWVersion,
			Name:         c.PanelType.String(),
			Version:      c.SWVersion,
		},
	}

	configRaw, _ := json.Marshal(configValue)
	client.Publish(configTopic, 0, true, configRaw)

	updateZone(z, client)
}

func updateZone(z *concord.Zone, client MQTT.Client) {
	client.Publish(stateTopic(*z), 0, true, stateValue(*z))
}

func setupSigHandler() chan bool {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info("Ctrl-C")
		done <- true
	}()
	return done
}
