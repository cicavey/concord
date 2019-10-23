package concord

import (
	"bytes"
	"testing"
)

func TestEncodeMessage(t *testing.T) {
	// 03 02 03 - zone info
	expected := []byte{SOM, 0x30, 0x33, 0x30, 0x32, 0x30, 0x33, 0x30, 0x38}
	result := encodeMessage([]byte{0x03, 0x02, 0x03})
	if bytes.Compare(expected, result) != 0 {
		t.Error("Messaging encoding failed")
	}
}
