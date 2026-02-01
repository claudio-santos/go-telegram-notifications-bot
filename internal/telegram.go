package internal

import (
	"fmt"
	"net/http"
	"strconv"
)

// TelegramService handles all Telegram-related functionality
type TelegramService struct {
	ConfigManager *ConfigManager
}

// NewTelegramService creates a new Telegram service
func NewTelegramService(cm *ConfigManager) *TelegramService {
	return &TelegramService{
		ConfigManager: cm,
	}
}

// SendTestTelegram sends a test message to Telegram
func (ts *TelegramService) SendTestTelegram(item map[string]interface{}, feed map[string]interface{}) error {
	token := ts.ConfigManager.Config.TestTelegramApiToken
	chatID := ts.ConfigManager.Config.TestTelegramChatId
	threadID := ts.ConfigManager.Config.TestTelegramMessageThreadId
	template := ts.ConfigManager.Config.TestTelegramTemplate

	if token == "" {
		return fmt.Errorf("test Telegram API token not configured")
	}

	if chatID == 0 {
		return fmt.Errorf("test Telegram chat ID not configured")
	}

	if template == "" {
		template = "{{.Title}}"
	}

	message := ProcessFeedItemForTelegram(item, feed, template)

	telegramMsg := TelegramMessage{
		ChatID:          chatID,
		Text:            message,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	}

	return SendTelegramMessage(token, telegramMsg)
}

// SendFeedItemToTelegram sends a feed item to Telegram based on the feed configuration
func (ts *TelegramService) SendFeedItemToTelegram(feed Feed, item map[string]interface{}) error {
	token := feed.TelegramApiToken
	chatID := feed.TelegramChatId
	threadID := feed.TelegramMessageThreadId
	template := feed.TelegramTemplate

	if token == "" || chatID == 0 {
		return fmt.Errorf("Telegram configuration is incomplete for feed: %s", feed.FeedUrl)
	}

	if template == "" {
		template = "{{.Title}}"
	}

	feedMap := map[string]interface{}{
		"Title":       "",
		"Description": "",
		"Link":        feed.FeedUrl,
		"Language":    "",
		"Copyright":   "",
		"Generator":   "",
		"FeedType":    "",
		"FeedVersion": "",
	}

	message := ProcessFeedItemForTelegram(item, feedMap, template)

	telegramMsg := TelegramMessage{
		ChatID:          chatID,
		Text:            message,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	}

	err := SendTelegramMessage(token, telegramMsg)
	if err != nil {
		return fmt.Errorf("error sending feed item to Telegram: %v", err)
	}

	return nil
}

// HandleTestTelegramByIndex handles testing Telegram notifications by retrieving the item from global storage using its index
func (ts *TelegramService) HandleTestTelegramByIndex(w http.ResponseWriter, r *http.Request) {
	itemIndexStr := r.FormValue("item_index")
	feedUrl := r.FormValue("feed_url")

	if itemIndexStr == "" {
		http.Error(w, "Item index is required", http.StatusBadRequest)
		return
	}

	index, err := strconv.Atoi(itemIndexStr)
	if err != nil {
		http.Error(w, "Invalid item index", http.StatusBadRequest)
		return
	}

	// Retrieve the item from global storage with thread safety
	tempFeedMutex.RLock()
	if index < 0 || index >= len(tempFeedItems) {
		tempFeedMutex.RUnlock()
		http.Error(w, "Item not found at the given index", http.StatusBadRequest)
		return
	}

	item := tempFeedItems[index]
	tempFeedMutex.RUnlock()

	// Create feed map with actual feed information from stored feed
	feedMap := map[string]interface{}{
		"Title":       "",
		"Description": "",
		"Link":        "",
		"Language":    "",
		"Copyright":   "",
		"Generator":   "",
		"FeedType":    "",
		"FeedVersion": "",
	}

	// Use the stored feed info if available
	tempFeedMutex.RLock()
	if tempFeedInfo != nil {
		feedMap["Title"] = tempFeedInfo.Title
		feedMap["Description"] = tempFeedInfo.Description
		feedMap["Link"] = tempFeedInfo.Link
		feedMap["Language"] = tempFeedInfo.Language
		feedMap["Copyright"] = tempFeedInfo.Copyright
		feedMap["Generator"] = tempFeedInfo.Generator
		feedMap["FeedType"] = tempFeedInfo.FeedType
		feedMap["FeedVersion"] = tempFeedInfo.FeedVersion
	}
	tempFeedMutex.RUnlock()

	err = ts.SendTestTelegram(item, feedMap)
	if err != nil {
		http.Error(w, "Error sending to Telegram: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if feedUrl != "" {
		http.Redirect(w, r, "/?url="+feedUrl, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
