package history

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const discordAttachmentPrefix = "https://cdn.discordapp.com/attachments/"

func DownloadAndSaveFile(rawURL string, saveDir string) (string, error) {
	decodedURL, err := url.QueryUnescape(rawURL)
	if err != nil {
		return "", fmt.Errorf("URLデコードに失敗しました: %w", err)
	}

	if !strings.HasPrefix(decodedURL, discordAttachmentPrefix) {
		return "", fmt.Errorf("URLではありません: %s", decodedURL)
	}

	resp, err := http.Head(decodedURL)
	if err != nil {
		return "", fmt.Errorf("HTTP HEADリクエストに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("(Content-Type: %s): %s", contentType, decodedURL)
	}

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			return "", fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
		}
	}

	parsedURL, err := url.Parse(decodedURL)
	if err != nil {
		return "", fmt.Errorf("URLパースに失敗しました: %w", err)
	}
	fileName := filepath.Base(parsedURL.Path)
	if fileName == "" || fileName == "/" {
		ext := ""
		if parts := strings.Split(contentType, "/"); len(parts) == 2 {
			ext = "." + parts[1]
		}
		fileName = fmt.Sprintf("download_%d%s", time.Now().UnixNano(), ext)
	} else {
		ext := filepath.Ext(fileName)
		base := strings.TrimSuffix(fileName, ext)
		fileName = fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
	}

	filePath := filepath.Join(saveDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの作成に失敗しました: %w", err)
	}
	defer file.Close()

	resp, err = http.Get(decodedURL)
	if err != nil {
		return "", fmt.Errorf("ファイルのダウンロードに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ファイルのダウンロードに失敗しました (ステータスコード: %d): %s", resp.StatusCode, decodedURL)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("ファイルの保存に失敗しました: %w", err)
	}

	return filePath, nil
}
