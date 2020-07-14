package main

import (
	"strings"

	"github.com/spf13/viper"
)

const DefaultConfig = `
device: /dev/ttyUSB0

log_level: 

homeassistant:
  status_topic: hass/status
  discover_base: homeassistant

mqtt:
  broker: "tcp://localhost:1883"
  client_id: concord
  username: username
  password: password
`

const ConfDiscoverBase = "homeassistant.discover_base"

func ResolveConfig() (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigType("yaml")
	// Read in default config from static string, can later be written out if needed
	err := v.ReadConfig(strings.NewReader(DefaultConfig))
	if err != nil {
		return nil, err
	}

	v.SetConfigName("concord")               // name of config file (without extension)
	v.AddConfigPath("/etc/concord/")         // path to look for the config file in
	v.AddConfigPath("$HOME/.concord")        // call multiple times to add many search paths
	v.AddConfigPath("$HOME/.config/concord") // call multiple times to add many search paths
	v.AddConfigPath(".")                     // optionally look for config in the working directory
	v.AddConfigPath("/")

	v.MergeInConfig()

	v.SetEnvPrefix("CONCORD")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// os.Setenv("CONCORD_MQTT_BROKER", "TEST")
	// os.Setenv("CONCORD_HOMEASSISTANT_STATUS_TOPIC", "poop")

	// os.Args = append(os.Args, "--mqtt.port=56")

	// pflag.Int("mqtt.port", 1883, "MQTT Broker Port")
	// pflag.Parse()

	// err = v.BindPFlags(pflag.CommandLine)
	// if err != nil {
	// 	return nil, err
	// }

	return v, nil

	// fmt.Println(v.GetString("mqtt.broker"))
	// fmt.Println(v.GetString("mqtt.username"))
	// fmt.Println(v.GetInt("mqtt.port"))

	// fmt.Println(v.GetString("homeassistant.status_topic"))
	// fmt.Println(v.AllSettings())
}
