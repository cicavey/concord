# Concord

Concord is a simple service, primarily designed to run on a Raspberry Pi Zero, that interfaces with a Concord 4 alarm panel through Interlogix/GE Concord 4 Automation Module interface (rs232).

The goal of the project is to create a read-only interface to the Concord 4 sensors (and only sensors) into Home Assistant via MQTT for trigging events and automations - no direct manipulation of the alarm itself.

## Why MQTT?

I prefer to use MQTT as a message bus for many home automation utilities as this allows more flexibility. It is true that there is a more native solution to get Concord 4 into Home Assitant. See https://github.com/JasonCarter80/concord232.

## Configuration

Under the hood concord uses Viper for configuration and concord is configured with reasonable defaults and the ability to override these defaults either through configuration file (yaml) or environment variables.

### Example (default) configuration file

Omitting any key below in your configuration file will result in using the default value (listed below)

```
device: /dev/ttyUSB0

# Default is INFO, other levels include DEBUG, ERROR, WARN, FATAL, TRACE. DEBUG is most often used.
log_level: INFO


# Status topic is for tracking when HA itself comes back online. Concord will rebpulish zone definitions at that time
# Discover base is where the Concord topics will start/base from
homeassistant:
  status_topic: hass/status
  discover_base: homeassistant

# Broker is a Go-style socket defition including scheme, host/ip, and port.
# SSL can be used but currently the Root CA and Client SSL are not supported.
# SSL connections will not verify cert chain (insecure no verify) until Root CA support is added
# Client id is the MQTT client id
# User/Pass only used if not blank
mqtt:
  broker: "tcp://localhost:1883"
  client_id: concord
  username:
  password:
```

### Configuration file search path

The file `concord.yml` will be searched in the following locations (in order):

```
/etc/concord
$HOME/.concord
$HOME/.config/concord
. (binary working dir)
/ (file system root, handy for containers)
```

### Enviornment variables

Alternatively, configuration can be specified via environment which can be useful for containerization or testing. Take any key from the above configuration file, prepend "CONCORD*", convert to uppercase, and delimit via * (underscore) to create matching environment. Environment will override config file.

Examples:

CONCORD_MQTT_BROKER=tcp://localhost:1883
CONCORD_DEVICE=/dev/serial/by-id/usb-Prolific_Technology_Inc.\_USB-Serial_Controller-if00-port0

## systemd

See unit definition in contrib
