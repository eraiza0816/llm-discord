package discord

import "github.com/bwmarrin/discordgo"

func splitToEmbedFields(text string) []*discordgo.MessageEmbedField {
	const maxFieldLength = 1024 // Discord の Embed Field Value の最大文字数
	var fields []*discordgo.MessageEmbedField

	if text == "" {
		return fields
	}

	for i := 0; i < len(text); i += maxFieldLength {
		end := i + maxFieldLength
		if end > len(text) {
			end = len(text)
		}
		chunk := text[i:end]
		field := &discordgo.MessageEmbedField{
			Name:   "",
			Value:  chunk,
			Inline: false,
		}
		fields = append(fields, field)
	}
	return fields
}

// TODO: 他のコマンドハンドラで共通化できる Embed 作成ロジックがあれば、ここにヘルパー関数を追加していく！