package main

import (
	"os"
	"testing"
)

func TestEnvFileExists(t *testing.T) {
	// .envファイルが存在するかテスト
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		t.Errorf(".env file does not exist")
	}
}

func TestJsonFileExists(t *testing.T) {
	// json/command.jsonファイルが存在するかテスト
	if _, err := os.Stat("json/command.json"); os.IsNotExist(err) {
		t.Errorf("json/command.json file does not exist")
	}
}
