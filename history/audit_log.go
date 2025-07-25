package history

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	auditLogPath      = "data/audit.jsonl"
	timestampFormat = "2006-01-02T15:04:05.000Z07:00"
)

type AuditLogEntry struct {
	Timestamp        string `json:"timestamp"`
	GuildID          string `json:"guild_id,omitempty"`
	ChannelID        string `json:"channel_id"`
	MessageID        string    `json:"message_id"`
	UserID           string    `json:"user_id"`
	UserName         string    `json:"user_name"`
	Content          string    `json:"content"`
	Attachments      []string  `json:"attachments,omitempty"`
	IsDeleted        bool      `json:"is_deleted,omitempty"`
	EventType        string    `json:"event_type"`
}

func InitAuditLog() error {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		if err := os.MkdirAll("data", 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
		log.Println("Created data directory for audit logs.")
	}
	return nil
}

func CloseAuditLog() {
	log.Println("Audit log (JSONL) closed.")
}

func writeAuditLogEntry(entry AuditLogEntry) error {
	file, err := os.OpenFile(auditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log entry to JSON: %w", err)
	}

	if _, err := file.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write audit log entry to file: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline to audit log file: %w", err)
	}
	return nil
}

func LogMessageCreate(messageID, channelID, guildID, userID, userName, content string, attachments []string, timestamp time.Time) error {
	entry := AuditLogEntry{
		Timestamp:   timestamp.UTC().Format(timestampFormat),
		GuildID:     guildID,
		ChannelID:   channelID,
		MessageID:   messageID,
		UserID:      userID,
		UserName:    userName,
		Content:     content,
		Attachments: attachments,
		EventType:   "create",
	}
	return writeAuditLogEntry(entry)
}

func LogMessageUpdate(messageID, content string, updateTimestamp time.Time) error {
	entry := AuditLogEntry{
		Timestamp: updateTimestamp.UTC().Format(timestampFormat),
		MessageID: messageID,
		Content:   content,
		EventType: "update",
	}
	return writeAuditLogEntry(entry)
}

func LogMessageDelete(messageID string, deletionTimestamp time.Time) error {
	entry := AuditLogEntry{
		Timestamp:        deletionTimestamp.UTC().Format(timestampFormat),
		MessageID:        messageID,
		IsDeleted:        true,
		EventType:        "delete",
	}
	return writeAuditLogEntry(entry)
}
