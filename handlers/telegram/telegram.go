package telegram

import (
	"encoding/json"
	"strconv"
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
	Token        string   `json:"token" mapstructure:"token"`
	AllowedUsers []string `json:"allowed_users" mapstructure:"allowed_users"`
	bot          *telebot.Bot
}

// New creates a telegram handler. useful for sending notifications about events
func New() *Telegram {
	var t Telegram
	err := config.GetStruct("telegram", &t)
	if err != nil {
		logger.Fatal().Err(err).Msg("Creation failed")
	}
	return &t
}

func (h *Telegram) Topics() []types.CommandType {
	return []types.CommandType{types.TelegramNotificationCommandType}
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
				return true
			}

			return false
		}),
	})
	if err != nil {
		logger.Err(err).Msgf("Setup failed")
		return err
	}
	go h.bot.Start()
	return nil
}

func (h *Telegram) sendMessage(content string) {
	if h.bot == nil {
		err := h.startBot()
		if err != nil {
			logger.Error().Msgf("Faled sending message: %v", content)
			return
		}
	}
	for _, id := range h.AllowedUsers {
		idAsInt, _ := strconv.Atoi(id)
		logger.Debug().Str("content", content).Str("user_id", id).Msg("Sending a message")
		_, err := h.bot.Send(telebot.ChatID(idAsInt), content)
		if err != nil {
			logger.Fatal().Err(err)
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

func (h *Telegram) Teardown() {
	h.bot.Stop()
}

func (h *Telegram) String() string {
	return "telegram"
}
