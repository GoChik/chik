package badge

import (
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/badge/rfid"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	"github.com/tarm/serial"
)

const name = "badge"

var logger = log.With().Str("handler", name).Logger()

type status map[string]types.TimeIndication

type conf struct {
	SerialPort string
	Identities map[string]string
}

type TagReader struct {
	chik.BaseHandler
	conf    *conf
	decoder rfid.Decoder
	status  *chik.StatusHolder
}

func New() chik.Handler {
	var c conf
	err := config.GetStruct(name, &c)
	if err != nil {
		logger.Warn().Msgf("Cannot get actions form config file: %v", err)
		config.Set(name, c)
	}
	return &TagReader{
		conf:   &c,
		status: chik.NewStatusHolder(name),
	}
}

func (r *TagReader) String() string {
	return name
}

func (r *TagReader) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	conf := &serial.Config{
		Name: r.conf.SerialPort,
		Baud: 19200,
	}
	logger.Debug().Msgf("Serial config: %v", conf)
	stream, err := serial.OpenPort(conf)
	if err != nil {
		return chik.Interrupts{}, err
	}
	r.decoder = &rfid.ICM522{stream}
	events := make(chan any, 1)
	go func() {
		for {
			tag, err := r.decoder.Decode()
			if err != nil {
				logger.Err(err).Msg("Decoding error")
				if errors.Is(err, io.EOF) {
					return
				}
				continue
			}
			logger.Debug().Msgf("Decoded tag: %v", tag)
			events <- tag
		}
	}()

	return chik.Interrupts{
		Timer: chik.NewEmptyTimer(),
		Event: events,
	}, nil
}

func (r *TagReader) HandleChannelEvent(event any, controller *chik.Controller) error {
	tag, ok := event.(*rfid.MiFareTag)
	if !ok {
		return errors.New("Unexpected event received")
	}
	tagId := hex.EncodeToString(tag.ID[:])
	identity := r.conf.Identities[tagId]
	if identity == "" {
		logger.Debug().Msgf("Tag %s not found", tagId)
		return nil
	}
	r.status.Edit(controller, func(current interface{}) (interface{}, bool) {
		s, ok := current.(status)
		if !ok {
			s = make(status)
		}
		s[identity] = types.TimeIndication(time.Now().Unix())
		return s, true
	})
	return nil
}

func (r *TagReader) Teardown() {
	r.decoder.Close()
}
