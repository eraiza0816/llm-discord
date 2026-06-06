package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/chat"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
)

// CommandHandler defines the interface for slash command handlers.
type CommandHandler interface {
	Name() string
	Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// commandDispatcher stores registered command handlers and dispatches interactions.
type commandDispatcher struct {
	handlers map[string]CommandHandler
}

func newCommandDispatcher() *commandDispatcher {
	return &commandDispatcher{
		handlers: make(map[string]CommandHandler),
	}
}

func (d *commandDispatcher) Register(h CommandHandler) {
	d.handlers[h.Name()] = h
}

func (d *commandDispatcher) Dispatch(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	name := i.ApplicationCommandData().Name
	h, ok := d.handlers[name]
	if !ok {
		return nil
	}
	return h.Handle(s, i)
}

// chatCommand implements the /chat command.
type chatCommand struct {
	chatSvc chat.Service
	cfg     *config.Config
}

func (c *chatCommand) Name() string { return "chat" }

func (c *chatCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	threadID := resolveThreadIDForInteraction(s, i)
	if threadID == "" {
		return nil
	}
	chatCommandHandler(s, i, c.chatSvc, threadID, c.cfg)
	return nil
}

// resetCommand implements the /reset command.
type resetCommand struct {
	historyMgr history.HistoryManager
}

func (c *resetCommand) Name() string { return "reset" }

func (c *resetCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	threadID := resolveThreadIDForInteraction(s, i)
	if threadID == "" {
		return nil
	}
	resetCommandHandler(s, i, c.historyMgr, threadID)
	return nil
}

// aboutCommand implements the /about command.
type aboutCommand struct {
	cfg *config.Config
}

func (c *aboutCommand) Name() string { return "about" }

func (c *aboutCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	aboutCommandHandler(s, i, c.cfg)
	return nil
}

// editCommand implements the /edit command.
type editCommand struct {
	cfg *config.Config
}

func (c *editCommand) Name() string { return "edit" }

func (c *editCommand) Handle(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	editCommandHandler(s, i, c.cfg)
	return nil
}

// resolveThreadIDForInteraction extracts the thread ID from an interaction.
func resolveThreadIDForInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	if i.ChannelID != "" {
		ch, err := s.State.Channel(i.ChannelID)
		if err != nil {
			ch, err = s.Channel(i.ChannelID)
			if err != nil {
				sendEphemeralErrorResponse(s, i, err)
				return ""
			}
		}
		if ch.IsThread() {
			return ch.ID
		}
		return i.ChannelID
	}

	if i.Message != nil && i.Message.ChannelID != "" {
		ch, err := s.State.Channel(i.Message.ChannelID)
		if err != nil {
			ch, err = s.Channel(i.Message.ChannelID)
			if err != nil {
				sendEphemeralErrorResponse(s, i, err)
				return ""
			}
		}
		if ch.IsThread() {
			return ch.ID
		}
		return i.Message.ChannelID
	}

	sendEphemeralErrorResponse(s, i, errNoThreadID)
	return ""
}
