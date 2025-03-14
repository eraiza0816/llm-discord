package main

import (
	"strings"
	"testing"
	"log"
)

func TestGetResponse(t *testing.T) {
	log.Println("Entering TestGetResponse function")
	defer log.Println("Exiting TestGetResponse function")
	chat, err := NewChat("test_token", "gemini-2.0-flash", "You are a helpful AI.")
	if err != nil {
		t.Fatalf("Failed to create chat instance: %v", err)
	}
	if chat != nil {
		defer chat.Close()
	}

	userID := "testuser"
	username := "TestUser"
	message := "Hello, Gemini!"
	timestamp := "2024-01-01T00:00:00Z"

	response, _, err := chat.GetResponse(userID, username, message, timestamp)
	if err != nil {
		t.Fatalf("GetResponse failed: %v", err)
	}

	if response == "" {
		t.Error("Response is empty")
	}

	if !strings.Contains(response, "Gemini") && !strings.Contains(response, "AI") {
		t.Errorf("Response does not contain expected content. Got: %s", response)
	}
}

func TestClearHistory(t *testing.T) {
	log.Println("Entering TestClearHistory function")
	defer log.Println("Exiting TestClearHistory function")
	chat, err := NewChat("test_token", "gemini-2.0-flash", "You are a helpful AI.")
	if err != nil {
		t.Fatalf("Failed to create chat instance: %v", err)
	}
	if chat != nil {
		defer chat.Close()
	}

	userID := "testuser"
	username := "TestUser"
	message := "Hello, Gemini!"
	timestamp := "2024-01-01T00:00:00Z"

	_, _, err = chat.GetResponse(userID, username, message, timestamp)
	if err != nil {
		t.Fatalf("GetResponse failed: %v", err)
	}

	chat.ClearHistory(userID)

	if len(chat.userHistories[userID]) != 0 {
		t.Error("Chat history was not cleared")
	}
}

func TestResetCommand(t *testing.T) {
	log.Println("Entering TestResetCommand function")
	defer log.Println("Exiting TestResetCommand function")
	chat, err := NewChat("test_token", "gemini-2.0-flash", "You are a helpful AI.")
	if err != nil {
		t.Fatalf("Failed to create chat instance: %v", err)
	}
	if chat != nil {
		defer chat.Close()
	}

	userID := "testuser"
	username := "TestUser"
	message := "/reset"
	timestamp := "2024-01-01T00:00:00Z"

	response, _, err := chat.GetResponse(userID, username, message, timestamp)
	if err != nil {
		t.Fatalf("GetResponse failed: %v", err)
	}

	if response != "チャット履歴をリセットしました！" {
		t.Errorf("Reset command failed. Got: %s, expected: チャット履歴をリセットしました！", response)
	}

	if len(chat.userHistories[userID]) != 0 {
		t.Error("Chat history was not cleared after reset command")
	}
}
