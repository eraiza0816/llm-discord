package discord

import "github.com/bwmarrin/discordgo"

// DiscordSession defines the interface for discordgo session methods used by the handlers.
// This allows for mocking the session in tests.
type DiscordSession interface {
	ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendReply(channelID string, content string, reference *discordgo.MessageReference, options ...discordgo.RequestOption) (*discordgo.Message, error)
	StateChannel(channelID string) (*discordgo.Channel, error)
	Channel(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
}

// discordgoSession is a wrapper around discordgo.Session to add the missing methods.
type discordgoSession struct {
	*discordgo.Session
}

func (s *discordgoSession) StateChannel(channelID string) (*discordgo.Channel, error) {
	return s.State.Channel(channelID)
}

// ensure discordgoSession implements DiscordSession
var _ DiscordSession = (*discordgoSession)(nil)

// ensure discordgo.Session implements parts of DiscordSession that it can
var _ interface {
	ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendReply(channelID string, content string, reference *discordgo.MessageReference, options ...discordgo.RequestOption) (*discordgo.Message, error)
	Channel(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
} = (*discordgo.Session)(nil)
