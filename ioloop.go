package concord

import (
	"bufio"
	"encoding/hex"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

func (c *Client) ioLoop() {
	reader := bufio.NewReader(c.io)

	var msgCount int64

	for {
		buffer := make([]byte, 516)

		n, _ := reader.Read(buffer[:1])
		if n > 0 {

			// Check for start of message
			if buffer[0] != SOM {
				log.Debugf("! NO SOM: %x", buffer[0])
				continue
			}

			io.ReadFull(reader, buffer[1:3])
			lenB := make([]byte, 1)
			// length in bytes
			hex.Decode(lenB, buffer[1:3])
			// length in nibbles
			dataLen := lenB[0] * 2
			io.ReadFull(reader, buffer[3:3+dataLen])
			raw := make([]byte, lenB[0])
			hex.Decode(raw, buffer[3:3+dataLen])

			// Checksum
			cs := lenB[0]
			for _, v := range raw[:len(raw)-1] {
				cs += v
			}

			if cs != raw[len(raw)-1] {
				// invalid Checksum
				log.Warnf("---> invalid sum")
				c.io.Write([]byte{NAK})
				continue
			}

			// ACK
			c.io.Write([]byte{ACK})

			// Shave off checksum
			raw = raw[:len(raw)-1]

			msgCount++

			// Process message
			switch raw[0] {
			case 0x03: // Send Equipment List - Zone Data
				zoneID := int(raw[5])
				status := raw[7]
				name := "<unnamed>"
				if len(raw) >= 9 {
					name = decodeTokens(raw[8:])
				}

				// TODO: Store more
				zone := &Zone{
					ID:         zoneID,
					Name:       name,
					Status:     status,
					LastUpdate: time.Now(),
				}

				// TODO handle duplicate events - zone already created
				c.ZoneMap[zoneID] = zone

				c.eventQueue <- Event{Type: 0, Zone: zone}

				log.Printf("Zone List: %d: %s, status=%d", zoneID, name, status)
			case 0x21: // Zone Status
				zoneID := int(raw[4])
				status := raw[5]
				zone, ok := c.ZoneMap[zoneID]
				if !ok { // zone not initalized yet, ignore these events
					log.Warnf("Zone Status: UNSOL zone=%d, status=%d", zoneID, status)
					continue
				}

				log.Infof("Zone Status: %s, zone=%d, old=%d, new=%d", zone.Name, zone.ID, zone.Status, status)

				zone.Status = status
				zone.LastUpdate = time.Now()

				c.eventQueue <- Event{Type: 1, Zone: zone}

			case 0x22:
				switch raw[1] {
				case 0x05:
					// Siren Synchronize
					log.Debugln("-> Siren Synchronize")
				case 0x09:
					// Touchpad Display
					log.Debugln("-> Touchpad Display: part=%d, area=%d, type=%d, text=\n%s", raw[2], raw[3], raw[4], decodeTokens(raw[5:]))
				default:
					log.Debugf("? % X", raw)
				}
			default:
				log.Debugf("? % X", raw)
			}
		} else {

			select {
			case currentMsg := <-c.writeQueue:
				log.Debugf("W: % X", currentMsg)
				c.io.Write(currentMsg)
				c.io.Flush()
			default:
			}
		}
	}
}

// precomputed message "constants"
var msgDynRefresh = encodeMessage([]byte{0x02, 0x20, 0x22})
var msgEquipList = encodeMessage([]byte{0x02, 0x02, 0x04})
var msgZoneInfo = encodeMessage([]byte{0x03, 0x02, 0x03})

func encodeMessage(data []byte) []byte {
	// Output size is 2*data + 3 (SOM + 2 checksum)
	out := make([]byte, 2*len(data)+3)
	out[0] = SOM
	chk := checksum(data)
	hex.Encode(out[1:], data)
	hex.Encode(out[len(out)-2:], []byte{chk})
	return out
}

func checksum(data []byte) byte {
	var cs byte
	for _, v := range data {
		cs += v
	}
	return cs
}
