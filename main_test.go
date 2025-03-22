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
	// json/model.jsonファイルが存在するかテスト
	if _, err := os.Stat("json/model.json"); os.IsNotExist(err) {
		t.Errorf("json/model.json file does not exist")
	}
}
