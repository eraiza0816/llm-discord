package discord_test

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/eraiza0816/llm-discord/discord"
)

func TestSplitToEmbedFields(t *testing.T) {
	const maxLen = 1024 // discordgo.MessageEmbedFieldValueCharacterLimit

	tests := []struct {
		name           string
		inputText      string
		expectedNumFields int
		expectedValues []string
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
			name:              "String with multi-byte characters slightly over",
			inputText:         strings.Repeat("あ", 1024) + "い",
			expectedNumFields: 2,
			expectedValues:    []string{strings.Repeat("あ", 1024), "い"},
		},
		{
			name:              "String with multi-byte characters exactly 2 * maxLen",
			inputText:         strings.Repeat("あ", 1024) + strings.Repeat("い", 1024),
			expectedNumFields: 2,
			expectedValues:    []string{strings.Repeat("あ", 1024), strings.Repeat("い", 1024)},
		},
		{
			name:              "String exceeding total length limit",
			inputText:         strings.Repeat("a", 1024) + strings.Repeat("b", 1024) + strings.Repeat("c", 1024),
			expectedNumFields: 3,
			expectedValues: []string{
				strings.Repeat("a", 1024),
				strings.Repeat("b", 1024),
				strings.Repeat("c", 3000-1024-1024-3) + "...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fields []*discordgo.MessageEmbedField
			fields = discord.SplitToEmbedFields(tt.inputText)

			if len(fields) != tt.expectedNumFields {
				t.Errorf("SplitToEmbedFields(%q) returned %d fields, want %d", tt.inputText, len(fields), tt.expectedNumFields)
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
			if strings.HasPrefix(tt.name, "String with multi-byte characters") && len(fields) > 0 {
				if !strings.HasPrefix(fields[0].Value, "あ") {
					t.Errorf("Multi-byte test case: First field should start with 'あ', but got %q", fields[0].Value[:10]+"...")
				}
			}
		})
	}
}
