package discord

import (
	"errors"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/config"
	"github.com/eraiza0816/llm-discord/history"
	"github.com/eraiza0816/llm-discord/loader"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
)

// MockChatService for testing
type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) GetResponse(userID, threadID, username, message, timestamp, prompt string) (string, float64, string, error) {
	args := m.Called(userID, threadID, username, message, timestamp, prompt)
	return args.String(0), args.Get(1).(float64), args.String(2), args.Error(3)
}

func (m *MockChatService) Close() {
	m.Called()
}

// MockHistoryManager for testing
type MockHistoryManager struct {
	mock.Mock
}

func (m *MockHistoryManager) Add(userID, threadID, message, response string) error {
	args := m.Called(userID, threadID, message, response)
	return args.Error(0)
}

func (m *MockHistoryManager) Get(userID, threadID string) ([]history.HistoryMessage, error) {
	args := m.Called(userID, threadID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]history.HistoryMessage), args.Error(1)
}

func (m *MockHistoryManager) Clear(userID, threadID string) error {
	args := m.Called(userID, threadID)
	return args.Error(0)
}

func (m *MockHistoryManager) ClearAllByThreadID(threadID string) error {
	args := m.Called(threadID)
	return args.Error(0)
}

func (m *MockHistoryManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockDiscordSession for testing
type MockDiscordSession struct {
	mock.Mock
}

func (m *MockDiscordSession) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	args := m.Called(channelID, content)
	return args.Get(0).(*discordgo.Message), args.Error(1)
}

func (m *MockDiscordSession) ChannelMessageSendReply(channelID, content string, reference *discordgo.MessageReference, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	args := m.Called(channelID, content, reference)
	return args.Get(0).(*discordgo.Message), args.Error(1)
}

func (m *MockDiscordSession) StateChannel(channelID string) (*discordgo.Channel, error) {
	args := m.Called(channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*discordgo.Channel), args.Error(1)
}

func (m *MockDiscordSession) Channel(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	args := m.Called(channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*discordgo.Channel), args.Error(1)
}

func TestHandleMessageEvent(t *testing.T) {
	// Common setup
	mockCfg := &config.Config{
		Model: &loader.ModelConfig{
			Prompts: map[string]string{"default": "default prompt"},
		},
	}

	// Suppress log output during tests
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
	})

	t.Run("DM message", func(t *testing.T) {
		mockChatSvc := new(MockChatService)
		mockSession := new(MockDiscordSession)
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg_id",
				ChannelID: "dm_channel_id",
				GuildID:   "", // DM
				Author:    &discordgo.User{ID: "user_id", Username: "user"},
				Content:   "hello",
				Timestamp: time.Now(),
			},
		}
		mockChatSvc.On("GetResponse", "user_id", "dm_channel_id", "user", "hello", mock.Anything, "default prompt").Return("response", 1.0, "model", nil).Once()
		mockSession.On("ChannelMessageSend", "dm_channel_id", "response").Return(&discordgo.Message{}, nil).Once()

		handleMessageEvent(mockSession, m, mockChatSvc, mockCfg, MessageTypeDM, "dm_channel_id")

		mockChatSvc.AssertExpectations(t)
		mockSession.AssertExpectations(t)
	})

	t.Run("Reply to bot", func(t *testing.T) {
		mockChatSvc := new(MockChatService)
		mockSession := new(MockDiscordSession)
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg_id",
				ChannelID: "channel_id",
				GuildID:   "guild_id",
				Author:    &discordgo.User{ID: "user_id", Username: "user"},
				Content:   "hello again",
				Timestamp: time.Now(),
				ReferencedMessage: &discordgo.Message{
					Author: &discordgo.User{ID: "bot_id"},
				},
			},
		}
		mockChatSvc.On("GetResponse", "user_id", "thread_id", "user", "hello again", mock.Anything, "default prompt").Return("response", 1.0, "model", nil).Once()
		mockSession.On("ChannelMessageSendReply", "channel_id", "response", m.Reference()).Return(&discordgo.Message{}, nil).Once()

		handleMessageEvent(mockSession, m, mockChatSvc, mockCfg, MessageTypeReply, "thread_id")

		mockChatSvc.AssertExpectations(t)
		mockSession.AssertExpectations(t)
	})

	t.Run("Ignore self message", func(t *testing.T) {
		mockChatSvc := new(MockChatService)
		mockSession := new(MockDiscordSession)
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{Author: &discordgo.User{ID: "bot_id"}},
		}

		handleMessageEvent(mockSession, m, mockChatSvc, mockCfg, MessageTypeSelf, "any_id")

		mockChatSvc.AssertNotCalled(t, "GetResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Normal message (log only)", func(t *testing.T) {
		mockChatSvc := new(MockChatService)
		mockSession := new(MockDiscordSession)
		m := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "msg_id",
				ChannelID: "channel_id",
				GuildID:   "guild_id",
				Author:    &discordgo.User{ID: "user_id", Username: "user"},
				Content:   "just a message",
				Timestamp: time.Now(),
			},
		}
		os.MkdirAll("data", 0755)
		defer os.RemoveAll("data")

		handleMessageEvent(mockSession, m, mockChatSvc, mockCfg, MessageTypeNormal, "channel_id")

		mockChatSvc.AssertNotCalled(t, "GetResponse", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestResolveThreadID(t *testing.T) {
	t.Run("Channel is a thread", func(t *testing.T) {
		mockSession := new(MockDiscordSession)
		threadChannel := &discordgo.Channel{ID: "thread_id", Type: discordgo.ChannelTypeGuildPublicThread}
		mockSession.On("StateChannel", "channel_id").Return(threadChannel, nil).Once()

		threadID := resolveThreadID(mockSession, "channel_id")

		assert.Equal(t, "thread_id", threadID)
		mockSession.AssertExpectations(t)
	})

	t.Run("Channel is not a thread", func(t *testing.T) {
		mockSession := new(MockDiscordSession)
		regularChannel := &discordgo.Channel{ID: "channel_id", Type: discordgo.ChannelTypeGuildText}
		mockSession.On("StateChannel", "channel_id").Return(regularChannel, nil).Once()

		threadID := resolveThreadID(mockSession, "channel_id")

		assert.Equal(t, "channel_id", threadID)
		mockSession.AssertExpectations(t)
	})

	t.Run("Channel not in state, fallback to API", func(t *testing.T) {
		mockSession := new(MockDiscordSession)
		threadChannel := &discordgo.Channel{ID: "thread_id", Type: discordgo.ChannelTypeGuildPublicThread}
		mockSession.On("StateChannel", "channel_id").Return(nil, errors.New("not in state")).Once()
		mockSession.On("Channel", "channel_id").Return(threadChannel, nil).Once()

		threadID := resolveThreadID(mockSession, "channel_id")

		assert.Equal(t, "thread_id", threadID)
		mockSession.AssertExpectations(t)
	})
}
