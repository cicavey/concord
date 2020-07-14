package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func TestReadConfig(t *testing.T) {

	os.Setenv("CONCORD_MQTT_BROKER", "tcp://newvalyria:1883")

	config, _ = ResolveConfig()

	fmt.Println(config.AllSettings())

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
		t.Error(token.Error())
	}

	time.Sleep(2 * time.Second)

	client.Disconnect(250)
}
