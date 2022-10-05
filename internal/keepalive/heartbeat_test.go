package keepalive_test

import (
	"log"
	"testing"
	"time"
	"xmpp/internal/keepalive"
)

func TestNewHeartBeat(t *testing.T) {
	log.Println("HeartBeat START >>>>>>>>>>>")
	hb := keepalive.New(5, 3)

	go hb.Start()
	time.Sleep(time.Second*6)
	hb.Beating()
	go hb.Stop()
	time.Sleep(time.Second*5)
	if <-hb.IsStop {
		log.Println("HeartBeat STOP >>>>>>>>>>>")
	}
}
