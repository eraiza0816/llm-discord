package discord

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestSplitToEmbedFields(t *testing.T) {
	const maxLen = 1024 // discordgo.MessageEmbedFieldValueCharacterLimit

	tests := []struct {
		name           string
		inputText      string
		expectedNumFields int
		expectedValues []string // 各フィールドの期待される値（最初の数フィールドをチェック）
	}{
		{
			name:           "Empty string",
			inputText:      "",
			expectedNumFields: 0,
			expectedValues: []string{},
		},
		{
			name:           "Short string (less than maxLen)",
			inputText:      "This is a short string.",
			expectedNumFields: 1,
			expectedValues: []string{"This is a short string."},
		},
		{
			name:           "String exactly maxLen",
			inputText:      strings.Repeat("a", maxLen),
			expectedNumFields: 1,
			expectedValues: []string{strings.Repeat("a", maxLen)},
		},
		{
			name:           "String slightly longer than maxLen",
			inputText:      strings.Repeat("b", maxLen) + "c",
			expectedNumFields: 2,
			expectedValues: []string{strings.Repeat("b", maxLen), "c"},
		},
		{
			name:           "String exactly 2 * maxLen",
			inputText:      strings.Repeat("x", maxLen) + strings.Repeat("y", maxLen),
			expectedNumFields: 2,
			expectedValues: []string{strings.Repeat("x", maxLen), strings.Repeat("y", maxLen)},
		},
		{
			name:           "String longer than 2 * maxLen",
			inputText:      strings.Repeat("1", maxLen) + strings.Repeat("2", maxLen) + "333",
			expectedNumFields: 3,
			expectedValues: []string{strings.Repeat("1", maxLen), strings.Repeat("2", maxLen), "333"},
		},
		{
			name:           "String with multi-byte characters",
			inputText:      strings.Repeat("あ", maxLen/3) + strings.Repeat("い", maxLen/3) + strings.Repeat("う", maxLen/3) + "え", // maxLen を超えるように調整
			expectedNumFields: 2, // バイト数ではなく文字数で分割されるはずなので、maxLen文字ずつ区切られる
			// 期待値の計算は少し複雑になるので、フィールド数と最初のフィールドの開始文字だけ確認
			// expectedValues: []string{strings.Repeat("あ", maxLen/3) + strings.Repeat("い", maxLen/3) + ... }, // 正確な値は省略
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := splitToEmbedFields(tt.inputText)

			if len(fields) != tt.expectedNumFields {
				t.Errorf("splitToEmbedFields(%q) returned %d fields, want %d", tt.inputText, len(fields), tt.expectedNumFields)
			}

			// フィールドの値もチェック（期待値が設定されている場合）
			for i, expectedVal := range tt.expectedValues {
				if i >= len(fields) {
					t.Errorf("Field index %d out of bounds (only %d fields returned)", i, len(fields))
					break
				}
				if fields[i].Value != expectedVal {
					// 長すぎる場合は省略して表示
					displayExpected := expectedVal
					displayGot := fields[i].Value
					if len(displayExpected) > 50 { displayExpected = displayExpected[:50] + "..." }
					if len(displayGot) > 50 { displayGot = displayGot[:50] + "..." }
					t.Errorf("Field %d value mismatch:\ngot:  %q\nwant: %q", i, displayGot, displayExpected)
				}
				// Name が空であること、Inline が false であることも確認
				if fields[i].Name != "" {
					t.Errorf("Field %d Name should be empty, got %q", i, fields[i].Name)
				}
				if fields[i].Inline != false {
					t.Errorf("Field %d Inline should be false, got %v", i, fields[i].Inline)
				}
			}

			// マルチバイト文字のテストケースの追加チェック（最初のフィールドの開始文字）
			if tt.name == "String with multi-byte characters" && len(fields) > 0 {
				if !strings.HasPrefix(fields[0].Value, "あ") {
					t.Errorf("Multi-byte test case: First field should start with 'あ', but got %q", fields[0].Value[:10]+"...")
				}
				if tt.expectedNumFields > 1 && len(fields) > 1 {
