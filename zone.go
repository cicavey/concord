package concord

import (
	"time"
)

type Zone struct {
	ID         int
	Name       string
	Status     byte
	LastUpdate time.Time
}
