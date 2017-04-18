package plugins

import (
	"os/exec"
	"time"

	"github.com/beevik/ntp"
)

type systemTime struct {
	ticker *time.Ticker
	stop   chan bool
}

// NewSystemTimePlugin creates a Plugin that is used to
// sync the system clock to ntp time.
//
// Clock is synchronized once per week
func NewSystemTimePlugin() Plugin {
	return &systemTime{
		stop: make(chan bool, 1),
	}
}

func (p *systemTime) Start() {
	p.ticker = time.NewTicker(24 * 7 * time.Hour)
	go func() {
		for {
			select {
			case <-p.ticker.C:
				date, err := ntp.Time("0.pool.ntp.org")
				if err == nil {
					cmd := exec.Command("date", "-s", date.Format("2006.01.02-15:04:05"))
					cmd.Run()
				}
				break

			case <-p.stop:
				return
			}
		}
	}()
}

func (p *systemTime) Stop() {
	p.ticker.Stop()
	p.stop <- true
	close(p.stop)
}
