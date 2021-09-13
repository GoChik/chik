package telegram

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/types"
	"github.com/rs/zerolog/log"
	"github.com/thoas/go-funk"
	"gopkg.in/tucnak/telebot.v2"
)

var logger = log.With().Str("handler", "telegram").Logger()
var retryInterval = 20 * time.Second

// Message is the message the bot can send
// it needs to be of type: TelegramNotificationCommandType
type Message struct {
	Message string `json:"message"`
}

// Telegram handler
type Telegram struct {
	chik.BaseHandler
	Token         string            `json:"token" mapstructure:"token"`
	AllowedUsers  []string          `json:"allowed_users" mapstructure:"allowed_users"`
	DevicesByName map[string]string `json:"devices_by_name" mapstructure:"devices_by_name"`
	SetStrings    []string          `json:"set_strings"`
	ResetStrings  []string          `json:"reset_strings"`
	bot           *telebot.Bot
	notifications chan interface{}
}

// New creates a telegram handler. useful for sending notifications about events
func New() *Telegram {
	var t Telegram
	err := config.GetStruct("telegram", &t)
	if err != nil {
		logger.Fatal().Err(err).Msg("Creation failed")
	}
	t.notifications = make(chan interface{})
	return &t
}

func (h *Telegram) Topics() []types.CommandType {
	return []types.CommandType{types.TelegramNotificationCommandType}
}

func findWord(text string, candidates []string) bool {
	for _, token := range strings.Split(text, " ") {
		for _, candidate := range candidates {
			if strings.ToLower(token) == strings.ToLower(candidate) {
				return true
			}
		}
	}
	return false
}

func (h *Telegram) startBot() error {
	var err error
	h.bot, err = telebot.NewBot(telebot.Settings{
		Token: h.Token,
		Poller: telebot.NewMiddlewarePoller(&telebot.LongPoller{Timeout: 10 * time.Second}, func(upd *telebot.Update) bool {
			if upd.Message == nil {
				return true
			}

			if funk.InStrings(h.AllowedUsers, strconv.Itoa(upd.Message.Sender.ID)) {
				logger.Debug().Msg("Sender allowed to communicate")
				return true
			}

			logger.Debug().Msgf("message not allowed: %v", upd)
			return false
		}),
	})

	h.bot.Handle(telebot.OnText, func(m *telebot.Message) {
		applyAction := func(action types.Action) {
			for _, token := range strings.Split(m.Text, " ") {
				if val, ok := h.DevicesByName[strings.ToLower(token)]; ok {
					h.notifications <- types.DigitalCommand{ApplianceID: val, Action: action}
				}
			}
		}

		if findWord(m.Text, h.SetStrings) {
			applyAction(types.SET)
			return
		}

		if findWord(m.Text, h.ResetStrings) {
			applyAction(types.RESET)
		}
	})

	if err != nil {
		logger.Err(err).Msgf("Setup failed")
		return err
	}
	go h.bot.Start()
	return nil
}

func (h *Telegram) Setup(controller *chik.Controller) (chik.Interrupts, error) {
	return chik.Interrupts{Timer: chik.NewStartupActionTimer(), Event: h.notifications}, nil
}

func (h *Telegram) sendMessage(content string) {
	for _, id := range h.AllowedUsers {
		idAsInt, _ := strconv.Atoi(id)
		logger.Debug().Str("content", content).Str("user_id", id).Msg("Sending a message")
		_, err := h.bot.Send(telebot.ChatID(idAsInt), content)
		if err != nil {
			logger.Err(err).Msg("failed sending message")
		}
	}
}

func (h *Telegram) HandleMessage(message *chik.Message, controller *chik.Controller) error {
	var notification Message
	err := json.Unmarshal(message.Command().Data, &notification)
	if err != nil {
		logger.Warn().Msg("Unexpected message")
		return nil
	}

	h.sendMessage(notification.Message)
	return nil
}

func (h *Telegram) HandleTimerEvent(tick time.Time, controller *chik.Controller) {
	h.startBot()
}

func (h *Telegram) HandleChannelEvent(event interface{}, controller *chik.Controller) {
	command, ok := event.(types.DigitalCommand)
	if !ok {
		logger.Error().Msg("Unexpected channel event")
	}
	controller.Pub(types.NewCommand(types.DigitalCommandType, command), chik.LoopbackID)
}

func (h *Telegram) Teardown() {
	h.bot.Stop()
}

func (h *Telegram) String() string {
	return "telegram"
}
