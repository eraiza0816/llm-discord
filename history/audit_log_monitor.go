package history

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

const (
	monitorInterval = 1 * time.Minute
	processedURLsFile = "data/processed_urls.json"
)

var (
	processedURLs     map[string]struct{}
	processedURLsLock sync.Mutex
	monitorStopChan   chan struct{}
	monitorWg         sync.WaitGroup
)

func init() {
	processedURLs = make(map[string]struct{})
	monitorStopChan = make(chan struct{})
	loadProcessedURLs()
}

func StartAuditLogMonitor(downloadDir string) {
	monitorWg.Add(1)
	go func() {
		defer monitorWg.Done()
		ticker := time.NewTicker(monitorInterval)
		defer ticker.Stop()
		processAuditLog(downloadDir)

		for {
			select {
			case <-ticker.C:
				processAuditLog(downloadDir)
			case <-monitorStopChan:
				saveProcessedURLs()
				return
			}
		}
	}()
}

func StopAuditLogMonitor() {
	close(monitorStopChan)
	monitorWg.Wait()
}

func processAuditLog(downloadDir string) {
	file, err := os.Open(auditLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry AuditLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		for _, attachmentURL := range entry.Attachments {
			processedURLsLock.Lock()
			_, found := processedURLs[attachmentURL]
			processedURLsLock.Unlock()

			if !found {
				_, err := DownloadAndSaveFile(attachmentURL, downloadDir)
				if err != nil {
				} else {
					processedURLsLock.Lock()
					processedURLs[attachmentURL] = struct{}{}
					processedURLsLock.Unlock()
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
	}
}

func loadProcessedURLs() {
	file, err := os.Open(processedURLsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var urls []string
	if err := decoder.Decode(&urls); err != nil {
		return
	}

	processedURLsLock.Lock()
	for _, u := range urls {
		processedURLs[u] = struct{}{}
	}
	processedURLsLock.Unlock()
}

func saveProcessedURLs() {
	processedURLsLock.Lock()
	defer processedURLsLock.Unlock()

	urls := make([]string, 0, len(processedURLs))
	for u := range processedURLs {
		urls = append(urls, u)
	}

	file, err := os.Create(processedURLsFile)
	if err != nil {
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(urls); err != nil {
	}
}
