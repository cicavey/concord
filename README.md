# Concord

Concord is a simple service, primarily designed to run on a Raspberry Pi Zero, that interfaces with a Concord 4 alarm panel through Interlogix/GE Concord 4 Automation Module interface (rs232).

The goal of the project is to create a read-only interface to the Concord 4 sensors (and only sensors) into Home Assistant via MQTT for trigging events and automations - no direct manipulation of the alarm itself.

## Why MQTT?

I prefer to use MQTT as a message bus for many home automation utilities as this allows more flexibility. It is true that there is a more native solution to get Concord 4 into Home Assitant. See https://github.com/JasonCarter80/concord232. 