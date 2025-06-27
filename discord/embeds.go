package discord

import "github.com/bwmarrin/discordgo"

// SplitToEmbedFields は、指定されたテキストをDiscordのEmbed Fieldの制約に合わせて分割します。
//
// 制約:
// - 各フィールドの値は `maxFieldLength` 文字以下。
// - フィールドの総数は `maxFields` 以下。
// - 全フィールドの合計文字数は `maxTotalLength` 以下。
//
// 分割ロジック:
// 1. テキストをルーン（文字）単位で処理します。
// 2. テキストが `maxFieldLength` を超える場合、`maxFieldLength` 文字ずつに分割し、それぞれを新しいフィールドとして追加します。
// 3. フィールド数や合計文字数の上限に達した場合、最後のフィールドの末尾に省略記号 "..." を付けて処理を終了します。
func SplitToEmbedFields(text string) []*discordgo.MessageEmbedField {
	const (
		maxFieldLength = 1024 // Discord の Embed Field Value の最大文字数
		maxTotalLength = 3000 // 全フィールドの合計文字数上限
		maxFields      = 5    // フィールド数の上限
		ellipsis       = "..."
		ellipsisLen    = len(ellipsis)
	)

	if text == "" {
		return nil
	}

	var fields []*discordgo.MessageEmbedField
	runes := []rune(text)
	currentTotalLength := 0

	for len(runes) > 0 && len(fields) < maxFields {
		// チャンクの長さを決定
		chunkLen := len(runes)
		if chunkLen > maxFieldLength {
			chunkLen = maxFieldLength
		}

		// チャンクを決定
		chunk := string(runes[:chunkLen])

		// フィールド数が上限に達し、かつまだ残りテキストがある場合
		isLastField := len(fields) == maxFields-1
		hasMoreText := len(runes) > chunkLen
		if isLastField && hasMoreText {
			// 省略記号を入れるためにチャンクを切り詰める
			chunk = string(runes[:maxFieldLength-ellipsisLen]) + ellipsis
			// これが最後のフィールドになる
			runes = nil
		} else {
			runes = runes[chunkLen:]
		}

		// 合計文字数制限のチェック
		if currentTotalLength+len([]rune(chunk)) > maxTotalLength {
			// 現在のチャンクを追加すると制限を超える
			allowedLen := maxTotalLength - currentTotalLength
			if allowedLen <= ellipsisLen {
				// 省略記号すら入らない場合は、ここで終了
				break
			}
			// 許容文字数に合わせてチャンクを切り詰めて省略記号を追加
			chunkRunes := []rune(chunk)
			chunk = string(chunkRunes[:allowedLen-ellipsisLen]) + ellipsis
			// これが最後のフィールドになる
			runes = nil
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Value: chunk,
		})
		currentTotalLength += len([]rune(chunk))

		// runesがnilに設定されたらループを抜ける
		if runes == nil {
			break
		}
	}

	return fields
}
