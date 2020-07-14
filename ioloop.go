package concord

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

type Command byte

const (
	cmdPanelType         Command = 0x01
	cmdZoneData                  = 0x03
	cmdPartData                  = 0x04
	cmdBusDevData                = 0x05
	cmdBusCapData                = 0x06
	cmdOutputData                = 0x07
	cmdEquipmentListDone         = 0x08
	cmdUserData                  = 0x09
	cmdScheduleData              = 0x0a
	cmdEventData                 = 0x0b
	cmdLightAttach               = 0x0c
	cmdZoneStatus                = 0x21
	cmdZoneStatusEx              = 0x22
)

type ZoneStatusSubCommand byte

const (
	zsSirenSynchronize ZoneStatusSubCommand = 0x05
	zsTouchPadDisplay  ZoneStatusSubCommand = 0x09
)

type PanelType byte

const (
	Concord                 PanelType = 0x14
	ConcordExpress                    = 0x0b
	ConcordExpress4                   = 0x1e
	ConcordEuro                       = 0x0e
	AdventCommercialFire250           = 0x0d
	AdventHomeNavigator132            = 0x0f
	AdventCommercialBurg250           = 0x10
	AdventHomeNavigator250            = 0x11
	AdventCommercialBurg500           = 0x15
	AdventCommercialFire500           = 0x16
	AdventCommercialFire132           = 0x17
	AdventCommercialBurg132           = 0x18
)

func isConcord(pt PanelType) bool {
	return pt == Concord || pt == ConcordExpress || pt == ConcordExpress4 || pt == ConcordEuro
}

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
			switch Command(raw[0]) {
			case cmdPanelType: // 01 14 07 01 40 92 01 45 10 76
				pt := PanelType(raw[1])

				hw := fmt.Sprintf("%d.%d", raw[2], raw[3])
				sw := fmt.Sprintf("%d.%d", raw[4], raw[5])

				if isConcord(pt) {
					letter := "?"
					if raw[2] > 0 && raw[2] < 27 {
						letter = string(64 + raw[2])
					}
					digit := "?"
					if raw[3] >= 0 && raw[3] <= 9 {
						digit = string(48 + raw[3])
					}
					hw = letter + digit
					sw = fmt.Sprintf("%d", int(raw[4])<<8+int(raw[5]))
				}

				sn := int(raw[6])<<24 + int(raw[7])<<16 + int(raw[8])<<8 + int(raw[9])

				c.PanelType = pt
				c.HWVersion = hw
				c.SWVersion = sw
				c.Serial = fmt.Sprintf("%d", sn)

				log.Printf("Panel Type: %s:%d, hw=%s, sw=%s, serial=%d", pt, pt, hw, sw, sn)

				c.eventQueue <- Event{Type: EventTypePanelDefined}

			case cmdZoneData: // Send Equipment List - Zone Data
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

				c.eventQueue <- Event{Type: EventTypeZoneDefined, Zone: zone}

				log.Printf("Zone List: %d: %s, status=%d", zoneID, name, status)
			case cmdZoneStatus: // Zone Status
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

				c.eventQueue <- Event{Type: EventTypeZoneUpdated, Zone: zone}

			case cmdEquipmentListDone:
				c.writeQueue <- msgDynRefresh

			case cmdZoneStatusEx:
				switch ZoneStatusSubCommand(raw[1]) {
				case zsSirenSynchronize:
					// Siren Synchronize
					log.Debugln("-> Siren Synchronize")
				case zsTouchPadDisplay:
					// Touchpad Display
					log.Debugf("-> Touchpad Display: part=%d, area=%d, type=%d, text=\n%s", raw[2], raw[3], raw[4], decodeTokens(raw[5:]))
				default:
					log.Debugf("ZS? % X", raw)
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
