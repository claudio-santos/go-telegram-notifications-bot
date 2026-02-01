package internal

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mmcdole/gofeed"
)

// Handlers manages all HTTP handlers
type Handlers struct {
	ConfigManager *ConfigManager
}

// NewHandlers creates a new Handlers instance
func NewHandlers(cm *ConfigManager) *Handlers {
	return &Handlers{
		ConfigManager: cm,
	}
}

// HomeHandler handles the home page
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
	tmpl.Execute(w, nil)
}

// PreviewHandler handles RSS feed preview
func (h *Handlers) PreviewHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	urlStr := r.FormValue("url")
	if urlStr == "" {
		// Re-render the index page with an error message
		data := map[string]interface{}{
			"Error": "URL is required",
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		tmpl.Execute(w, data)
		return
	}

	// Validate the URL
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		data := map[string]interface{}{
			"Error": "Invalid URL format",
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Check if it's a valid URL scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		data := map[string]interface{}{
			"Error": "URL must use http or https scheme",
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Parse the RSS feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(urlStr)
	if err != nil {
		data := map[string]interface{}{
			"Error": fmt.Sprintf("Failed to parse feed: %v", err),
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Limit to first 5 items
	if len(feed.Items) > 5 {
		feed.Items = feed.Items[:5]
	}

	// Prepare data for template
	data := map[string]interface{}{
		"Feed":  feed,
		"Items": feed.Items,
		"URL":   urlStr,
	}

	// Render the index page with the feed data
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
	tmpl.Execute(w, data)
}

// ConfigGetHandler handles getting the config page
func (h *Handlers) ConfigGetHandler(w http.ResponseWriter, r *http.Request) {
	// Check if we need to add an empty feed for the form
	addEmptyFeed := r.URL.Query().Get("add_feed") == "true"

	feeds := h.ConfigManager.Config.Feeds
	if addEmptyFeed {
		// Add an empty feed to the list
		feeds = append(feeds, Feed{})
	}

	data := map[string]interface{}{
		"Server":                      h.ConfigManager.Config.Server,
		"Database":                    h.ConfigManager.Config.Database,
		"TestTelegramApiToken":        h.ConfigManager.Config.TestTelegramApiToken,
		"TestTelegramChatId":          h.ConfigManager.Config.TestTelegramChatId,
		"TestTelegramMessageThreadId": h.ConfigManager.Config.TestTelegramMessageThreadId,
		"TestTelegramTemplate":        h.ConfigManager.Config.TestTelegramTemplate,
		"Feeds":                       feeds,
	}
	tmpl := template.Must(template.ParseFiles("templates/config.html", "templates/partials/navbar.html"))
	tmpl.Execute(w, data)
}

// ConfigPostHandler handles updating the config
func (h *Handlers) ConfigPostHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		// Render config page with error message
		data := map[string]interface{}{
			"Server":       h.ConfigManager.Config.Server,
			"Database":     h.ConfigManager.Config.Database,
			"Feeds":        h.ConfigManager.Config.Feeds,
			"ErrorMessage": "Error parsing form data: " + err.Error(),
		}
		tmpl := template.Must(template.ParseFiles("templates/config.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Create new config from form data
	newConfig := Config{
		Server:                      r.FormValue("server"),
		Database:                    r.FormValue("database"),
		TestTelegramApiToken:        r.FormValue("test_telegram_api_token"),
		TestTelegramChatId:          0, // Will be set below
		TestTelegramMessageThreadId: 0, // Will be set below
		TestTelegramTemplate:        r.FormValue("test_telegram_template"),
		Feeds:                       []Feed{},
	}

	// Parse the integer values
	if testChatIdStr := r.FormValue("test_telegram_chat_id"); testChatIdStr != "" {
		if testChatId, err := strconv.ParseInt(testChatIdStr, 10, 64); err == nil {
			newConfig.TestTelegramChatId = testChatId
		}
	}

	if testThreadIdStr := r.FormValue("test_telegram_message_thread_id"); testThreadIdStr != "" {
		if testThreadId, err := strconv.ParseInt(testThreadIdStr, 10, 64); err == nil {
			newConfig.TestTelegramMessageThreadId = testThreadId
		}
	}

	// Process feeds from form data
	feedUrls := r.Form["feed_urls"]
	feedIntervals := r.Form["feed_intervals"]
	feedRetentionDays := r.Form["feed_retention_days"]
	telegramTokens := r.Form["telegram_tokens"]
	telegramChatIds := r.Form["telegram_chat_ids"]
	telegramThreadIds := r.Form["telegram_thread_ids"]
	telegramTemplates := r.Form["telegram_templates"]

	// Create Feed structs from form arrays
	for i := 0; i < len(feedUrls); i++ {
		if feedUrls[i] != "" { // Only add if URL is provided
			interval := 30 // default value
			if i < len(feedIntervals) && feedIntervals[i] != "" {
				if val, err := strconv.Atoi(feedIntervals[i]); err == nil {
					interval = val
				}
			}

			retentionDays := 30 // default value
			if i < len(feedRetentionDays) && feedRetentionDays[i] != "" {
				if val, err := strconv.Atoi(feedRetentionDays[i]); err == nil {
					retentionDays = val
				}
			}

			chatId := int64(0) // default value
			if i < len(telegramChatIds) && telegramChatIds[i] != "" {
				if val, err := strconv.ParseInt(telegramChatIds[i], 10, 64); err == nil {
					chatId = val
				}
			}

			threadId := int64(0) // default value
			if i < len(telegramThreadIds) && telegramThreadIds[i] != "" {
				if val, err := strconv.ParseInt(telegramThreadIds[i], 10, 64); err == nil {
					threadId = val
				}
			}

			feed := Feed{
				FeedUrl:                  feedUrls[i],
				FeedFetchIntervalMinutes: interval,
				FeedRetentionDays:        retentionDays,
				TelegramApiToken:         "",
				TelegramChatId:           chatId,
				TelegramMessageThreadId:  threadId,
				TelegramTemplate:         "",
			}

			if i < len(telegramTokens) {
				feed.TelegramApiToken = telegramTokens[i]
			}
			if i < len(telegramTemplates) {
				feed.TelegramTemplate = telegramTemplates[i]
			}

			newConfig.Feeds = append(newConfig.Feeds, feed)
		}
	}

	// Update the global config
	h.ConfigManager.Config = &newConfig

	// Save to file
	err = h.ConfigManager.SaveConfig()
	if err != nil {
		// Render config page with error message
		data := map[string]interface{}{
			"Server":       newConfig.Server,
			"Database":     newConfig.Database,
			"Feeds":        newConfig.Feeds,
			"ErrorMessage": "Error saving config: " + err.Error(),
		}
		tmpl := template.Must(template.ParseFiles("templates/config.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Redirect to config page with success message
	http.Redirect(w, r, "/config", http.StatusSeeOther)
}

// SendTestTelegramHandler sends a feed item to Telegram for testing
func (h *Handlers) SendTestTelegramHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Get the item data from the form
	title := r.FormValue("title")
	description := r.FormValue("description")
	link := r.FormValue("link")

	if title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Create the item map
	item := map[string]interface{}{
		"Title":       title,
		"Description": description,
		"Link":        link,
	}

	// Get test configuration
	testToken := h.ConfigManager.Config.TestTelegramApiToken
	testChatID := h.ConfigManager.Config.TestTelegramChatId
	testThreadID := h.ConfigManager.Config.TestTelegramMessageThreadId
	testTemplate := h.ConfigManager.Config.TestTelegramTemplate

	if testToken == "" {
		http.Error(w, "Test Telegram API token not configured", http.StatusBadRequest)
		return
	}

	// Format the message using the test template
	message := testTemplate
	if message == "" {
		message = "{{.Title}}"
	}

	// Process the feed item for Telegram
	message = ProcessFeedItemForTelegram(item, message)

	// Create Telegram message
	telegramMsg := TelegramMessage{
		ChatID:          testChatID,
		Text:            message,
		ParseMode:       "HTML",
		MessageThreadID: testThreadID,
	}

	// Send to Telegram
	err = SendTelegramMessage(testToken, telegramMsg)
	if err != nil {
		http.Error(w, "Error sending to Telegram: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to the index page with a success message
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// SendFeedItemToTelegram sends a feed item to Telegram based on the feed configuration
func (h *Handlers) SendFeedItemToTelegram(feed Feed, item map[string]interface{}) error {
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

	// Process the feed item for Telegram
	message := ProcessFeedItemForTelegram(item, template)

	// Create Telegram message
	telegramMsg := TelegramMessage{
		ChatID:          chatID,
		Text:            message,
		ParseMode:       "HTML",
		MessageThreadID: threadID,
	}

	// Send to Telegram
	err := SendTelegramMessage(token, telegramMsg)
	if err != nil {
		return fmt.Errorf("error sending feed item to Telegram: %v", err)
	}

	return nil
}

// APICheckFeedHandler handles checking a feed via API
func (h *Handlers) APICheckFeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse feed: %v", err), http.StatusBadRequest)
		return
	}

	// Limit to first 5 items
	if len(feed.Items) > 5 {
		feed.Items = feed.Items[:5]
	}

	response := map[string]interface{}{
		"feed": map[string]interface{}{
			"title":           feed.Title,
			"description":     feed.Description,
			"link":            feed.Link,
			"feedLink":        feed.FeedLink,
			"links":           feed.Links,
			"updated":         feed.Updated,
			"updatedParsed":   feed.UpdatedParsed,
			"published":       feed.Published,
			"publishedParsed": feed.PublishedParsed,
			"author":          feed.Author,
			"authors":         feed.Authors,
			"language":        feed.Language,
			"image":           feed.Image,
			"copyright":       feed.Copyright,
			"generator":       feed.Generator,
			"categories":      feed.Categories,
			"custom":          feed.Custom,
			"feedType":        feed.FeedType,
			"feedVersion":     feed.FeedVersion,
		},
		"items": feed.Items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APIGetConfigHandler handles getting config via API
func (h *Handlers) APIGetConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.ConfigManager.Config)
}
