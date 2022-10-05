package keepalive

import (
	"log"
	"sync"
	"time"
)

type Heartbeat struct {
	ticker       *time.Ticker
	interval     time.Duration
	currentCount int
	maxCount     int
	mutex        *sync.Mutex
	IsStop       chan bool
}

func New(intervalSec, maxCount int) *Heartbeat {
	log.Printf("[hb] new heartbeat interval: %v, maxCount: %v\n", intervalSec, maxCount)
	hb := new(Heartbeat)
	hb.mutex = new(sync.Mutex)
	hb.currentCount = 0
	hb.interval = time.Second * time.Duration(intervalSec)
	hb.maxCount = maxCount
	hb.IsStop = make(chan bool)
	return hb
}

func (h *Heartbeat) Start() {
	log.Println("[hb] start heartbeat")
	h.ticker = time.NewTicker(h.interval)
	//defer h.Stop()
	for {
		<-h.ticker.C
		h.mutex.Lock()
		h.currentCount++
		h.mutex.Unlock()
		if h.currentCount >= h.maxCount {
			log.Printf("[hb] stop: %v/%v...\n", h.currentCount, h.maxCount)
			h.IsStop <- true
			return
		} else {
			log.Printf("[hb] check: %v/%v...wait(%v)\n", h.currentCount, h.maxCount, h.interval)
		}
	}
}

func (h *Heartbeat) Stop() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.ticker != nil {
		log.Println("[hb] stop heartbeat")
		h.ticker.Stop()
		close(h.IsStop)
	} else {
		log.Println("[hb] already stop heartbeat")
	}
}

func (h *Heartbeat) Beating() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.ticker != nil {
		log.Println("[hb] beating! current count reset!")
		h.ticker.Reset(h.interval)
		h.currentCount = 0
	} else {
		log.Println("[hb] can beating, already stop heartbeat")
	}
}
