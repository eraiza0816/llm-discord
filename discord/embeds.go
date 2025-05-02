package discord

import "github.com/bwmarrin/discordgo"

func splitToEmbedFields(text string) []*discordgo.MessageEmbedField {
	const maxFieldLength = 1024      // Discord の Embed Field Value の最大文字数
	const maxTotalLength = 5120      // 全フィールドの合計文字数上限 (Discordの6000より少し余裕を持たせる)
	const maxFields = 5              // フィールド数の上限
	const ellipsis = "..."
	const ellipsisLen = len(ellipsis)

	var fields []*discordgo.MessageEmbedField
	var currentTotalLength int

	if text == "" {
		return fields // 空の場合は空のスライスを返す
	}

	remainingText := text
	for len(remainingText) > 0 && len(fields) < maxFields {
		var chunk string
		chunkLength := 0
		runes := []rune(remainingText)
		remainingLength := len(runes)

		if remainingLength <= maxFieldLength {
			// 残り全てが1フィールドに収まる場合
			chunkLength = remainingLength
		} else {
			// 1024文字で区切る場合
			chunkLength = maxFieldLength
		}

		// 次のチャンクを追加すると合計文字数制限を超えるかチェック
		if currentTotalLength+chunkLength > maxTotalLength {
			// 制限を超える場合、追加可能な文字数でチャンクを切り詰める
			allowedLength := maxTotalLength - currentTotalLength
			if allowedLength <= ellipsisLen {
				// 省略記号すら入らない場合はループを終了
				break
			}
			chunkLength = allowedLength - ellipsisLen // 省略記号の分を引く
			// chunkLength 分のルーンを取り出して文字列に変換
			chunk = string(runes[:chunkLength]) + ellipsis
			remainingText = "" // ループを終了させる
		} else {
			// 制限を超えない場合
			// chunkLength 分のルーンを取り出して文字列に変換
			chunk = string(runes[:chunkLength])
			// 残りのルーンを文字列に変換
			remainingText = string(runes[chunkLength:])
		}

		// フィールド数が上限に達し、かつ残りテキストがある場合は、最後のチャンクに省略記号を追加
		if len(fields) == maxFields-1 && len(remainingText) > 0 {
			chunkRunes := []rune(chunk) // 現在のチャンクをルーンに変換
			currentChunkLen := len(chunkRunes)
			if currentChunkLen > maxFieldLength-ellipsisLen {
				// 省略記号を追加するために末尾を削る (ルーン単位で)
				chunk = string(chunkRunes[:maxFieldLength-ellipsisLen]) + ellipsis
			} else if currentChunkLen <= maxFieldLength { // 削る必要がない場合でも省略記号を追加
				chunk += ellipsis
			}
			// else のケース (currentChunkLen > maxFieldLength) は上の totalLength チェックで弾かれているはず
			remainingText = "" // ループを終了させる
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "", // フィールド名を空にする
			Value:  chunk,
			Inline: false,
		})
		currentTotalLength += len([]rune(chunk)) // runeの数でカウント

		// 合計文字数制限に達したらループ終了
		if currentTotalLength >= maxTotalLength {
			break
		}
	}

	return fields
}

