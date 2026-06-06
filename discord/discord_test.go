package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestExtractAttachmentURLs(t *testing.T) {
	t.Run("nil attachments", func(t *testing.T) {
		urls := extractAttachmentURLs(nil)
		if urls == nil {
			t.Error("Expected non-nil slice, got nil")
		}
		if len(urls) != 0 {
			t.Errorf("Expected 0 urls, got %d", len(urls))
		}
	})

	t.Run("empty attachments", func(t *testing.T) {
		urls := extractAttachmentURLs([]*discordgo.MessageAttachment{})
		if len(urls) != 0 {
			t.Errorf("Expected 0 urls, got %d", len(urls))
		}
	})

	t.Run("multiple attachments", func(t *testing.T) {
		attachments := []*discordgo.MessageAttachment{
			{URL: "https://cdn.discordapp.com/attachments/1/image.png"},
			{URL: "https://cdn.discordapp.com/attachments/2/file.pdf"},
		}
		urls := extractAttachmentURLs(attachments)
		if len(urls) != 2 {
			t.Errorf("Expected 2 urls, got %d", len(urls))
		}
		if urls[0] != "https://cdn.discordapp.com/attachments/1/image.png" {
			t.Errorf("Expected first URL, got %q", urls[0])
		}
		if urls[1] != "https://cdn.discordapp.com/attachments/2/file.pdf" {
			t.Errorf("Expected second URL, got %q", urls[1])
		}
	})
}

type mockSessionForClassify struct {
	DiscordSession
	selfUserID string
}

func (m *mockSessionForClassify) ownUserID() string {
	return m.selfUserID
}

func TestClassifyMessageType(t *testing.T) {
	t.Run("self message", func(t *testing.T) {
		s := &discordgo.Session{State: &discordgo.State{Ready: discordgo.Ready{User: &discordgo.User{ID: "bot_id"}}}}
		m := &discordgo.MessageCreate{Message: &discordgo.Message{Author: &discordgo.User{ID: "bot_id"}}}
		msgType := classifyMessageType(s, m)
		if msgType != MessageTypeSelf {
			t.Errorf("Expected MessageTypeSelf, got %v", msgType)
		}
	})

	t.Run("DM message (no GuildID)", func(t *testing.T) {
		s := &discordgo.Session{State: &discordgo.State{Ready: discordgo.Ready{User: &discordgo.User{ID: "bot_id"}}}}
		m := &discordgo.MessageCreate{Message: &discordgo.Message{
			Author:  &discordgo.User{ID: "user_id"},
			GuildID: "",
		}}
		msgType := classifyMessageType(s, m)
		if msgType != MessageTypeDM {
			t.Errorf("Expected MessageTypeDM, got %v", msgType)
		}
	})

	t.Run("reply to bot", func(t *testing.T) {
		s := &discordgo.Session{State: &discordgo.State{Ready: discordgo.Ready{User: &discordgo.User{ID: "bot_id"}}}}
		m := &discordgo.MessageCreate{Message: &discordgo.Message{
			Author:  &discordgo.User{ID: "user_id"},
			GuildID: "guild_id",
			ReferencedMessage: &discordgo.Message{
				Author: &discordgo.User{ID: "bot_id"},
			},
		}}
		msgType := classifyMessageType(s, m)
		if msgType != MessageTypeReply {
			t.Errorf("Expected MessageTypeReply, got %v", msgType)
		}
	})

	t.Run("normal message", func(t *testing.T) {
		s := &discordgo.Session{State: &discordgo.State{Ready: discordgo.Ready{User: &discordgo.User{ID: "bot_id"}}}}
		m := &discordgo.MessageCreate{Message: &discordgo.Message{
			Author:  &discordgo.User{ID: "user_id"},
			GuildID: "guild_id",
		}}
		msgType := classifyMessageType(s, m)
		if msgType != MessageTypeNormal {
			t.Errorf("Expected MessageTypeNormal, got %v", msgType)
		}
	})
}

func TestJapanStandardTime(t *testing.T) {
	jst := japanStandardTime()
	zone, offset := jst.Zone()
	if zone != "JST" {
		t.Errorf("Expected JST timezone, got %s", zone)
	}
	if offset != 9*60*60 {
		t.Errorf("Expected JST offset 32400, got %d", offset)
	}
}

func TestDiscordgoSessionImplements(t *testing.T) {
	var _ DiscordSession = (*discordgoSession)(nil)
	var _ interface {
		ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
		ChannelMessageSendReply(channelID string, content string, reference *discordgo.MessageReference, options ...discordgo.RequestOption) (*discordgo.Message, error)
		Channel(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	} = (*discordgo.Session)(nil)
}
