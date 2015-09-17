package netio

import (

	"testing"
	"github.com/bmizerany/assert"
)




func TestDecodePayload(t *testing.T) {
	raw := []byte(`�59�5:::{"name":"set_uuid","args":["06dcHVX6la+UWnyOifjEAg=="]}�67�5:::{"name":"set_uuid","args":["HR7aU6D72fRLroK3lMesKR9dEizWMV9q"]}`)
	
	
	assert.Equal(t, []byte("�"), []byte("\ufffd"))
	
	
	packets, err := decodePayload(raw)
	if err != nil {
		t.Error(err)
		return
	}

	for index, msg := range packets {
		t.Log(index,msg)
	}
}

func TestDecodePacket(t *testing.T) {
	raw := []byte(`5:::{"name":"set_uuid","args":["06dcHVX6la+UWnyOifjEAg=="]}`)
	packets, err := decodePayload(raw)
	if err != nil {
		t.Error(err)
		return
	}

	for index, msg := range packets {
		t.Log(index,msg)
	}
}
