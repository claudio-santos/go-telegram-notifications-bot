package internal

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/mmcdole/gofeed"
)

// Global variable to store feed items temporarily with thread-safe access
var (
	tempFeedItems []map[string]interface{}
	tempFeedMutex sync.RWMutex
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

// IndexGetHandler handles the home page
func (h *Handlers) IndexGetHandler(w http.ResponseWriter, r *http.Request) {
	// Check if there's a URL parameter to auto-preview
	urlStr := r.URL.Query().Get("url")
	if urlStr != "" {
		// Process the feed preview like a POST request would
		h.processFeedPreview(w, urlStr)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
	tmpl.Execute(w, nil)
}

// sanitizeFeedData sanitizes feed data to prevent XSS and other issues
func sanitizeFeedData(feed *gofeed.Feed) {
	// Sanitize feed-level fields
	feed.Title = SanitizeText(feed.Title)
	feed.Description = SanitizeText(feed.Description)
	feed.Link = SanitizeText(feed.Link)
	feed.FeedLink = SanitizeText(feed.FeedLink)
	feed.Language = SanitizeText(feed.Language)
	feed.Copyright = SanitizeText(feed.Copyright)
	feed.Generator = SanitizeText(feed.Generator)
	feed.FeedType = SanitizeText(feed.FeedType)
	feed.FeedVersion = SanitizeText(feed.FeedVersion)

	// Sanitize author information
	if feed.Author != nil {
		feed.Author.Name = SanitizeText(feed.Author.Name)
		feed.Author.Email = SanitizeText(feed.Author.Email)
	}

	// Sanitize authors
	for _, author := range feed.Authors {
		if author != nil {
			author.Name = SanitizeText(author.Name)
			author.Email = SanitizeText(author.Email)
		}
	}

	// Sanitize image information
	if feed.Image != nil {
		feed.Image.URL = SanitizeText(feed.Image.URL)
		feed.Image.Title = SanitizeText(feed.Image.Title)
	}

	// Sanitize items
	for _, item := range feed.Items {
		if item != nil {
			item.Title = SanitizeText(item.Title)
			item.Description = SanitizeText(item.Description)
			item.Content = SanitizeText(item.Content)
			item.Link = SanitizeText(item.Link)
			item.GUID = SanitizeText(item.GUID)
			item.Updated = SanitizeText(item.Updated)
			item.Published = SanitizeText(item.Published)

			// Sanitize item author
			if item.Author != nil {
				item.Author.Name = SanitizeText(item.Author.Name)
				item.Author.Email = SanitizeText(item.Author.Email)
			}

			// Sanitize item authors
			for _, author := range item.Authors {
				if author != nil {
					author.Name = SanitizeText(author.Name)
					author.Email = SanitizeText(author.Email)
				}
			}

			// Sanitize item image
			if item.Image != nil {
				item.Image.URL = SanitizeText(item.Image.URL)
				item.Image.Title = SanitizeText(item.Image.Title)
			}

			// Sanitize categories
			for i, category := range item.Categories {
				item.Categories[i] = SanitizeText(category)
			}

			// Sanitize links
			for i, link := range item.Links {
				item.Links[i] = SanitizeText(link)
			}

			// Sanitize enclosures
			for _, enclosure := range item.Enclosures {
				if enclosure != nil {
					enclosure.URL = SanitizeText(enclosure.URL)
					enclosure.Type = SanitizeText(enclosure.Type)
					enclosure.Length = SanitizeText(enclosure.Length)
				}
			}

			// Sanitize custom fields
			for k, v := range item.Custom {
				item.Custom[k] = SanitizeText(v)
			}
		}
	}
}

// processFeedPreview handles the actual feed preview logic
func (h *Handlers) processFeedPreview(w http.ResponseWriter, urlStr string) {
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

	// Sanitize feed data before passing to template
	sanitizeFeedData(feed)

	// Limit to first 5 items
	if len(feed.Items) > 5 {
		feed.Items = feed.Items[:5]
	}

	// Convert feed items to a format suitable for storage and assign indices
	var itemsForStorage []map[string]interface{}
	for _, item := range feed.Items {
		itemMap := map[string]interface{}{
			"Title":       item.Title,
			"Description": item.Description,
			"Content":     item.Content,
			"Link":        item.Link,
			"Updated":     item.Updated,
			"Published":   item.Published,
			"GUID":        item.GUID,
		}

		// Add author information if available
		if item.Author != nil {
			itemMap["Author"] = map[string]interface{}{
				"Name":  item.Author.Name,
				"Email": item.Author.Email,
			}
		} else {
			itemMap["Author"] = nil
		}

		// Add multiple authors if available
		if item.Authors != nil {
			var authorsList []interface{}
			for _, author := range item.Authors {
				if author != nil {
					authorsList = append(authorsList, map[string]interface{}{
						"Name":  author.Name,
						"Email": author.Email,
					})
				}
			}
			itemMap["Authors"] = authorsList
		}

		// Add categories if available
		if item.Categories != nil {
			itemMap["Categories"] = item.Categories
		}

		// Add image information if available
		if item.Image != nil {
			itemMap["Image"] = map[string]interface{}{
				"URL":   item.Image.URL,
				"Title": item.Image.Title,
			}
		}

		// Add links if available
		if item.Links != nil {
			itemMap["Links"] = item.Links
		}

		// Add updated parsed if available
		if item.UpdatedParsed != nil {
			itemMap["UpdatedParsed"] = item.UpdatedParsed.Format("2006-01-02 15:04:05 MST")
		}

		// Add published parsed if available
		if item.PublishedParsed != nil {
			itemMap["PublishedParsed"] = item.PublishedParsed.Format("2006-01-02 15:04:05 MST")
		}

		// Add enclosures if available
		if item.Enclosures != nil {
			var enclosuresList []interface{}
			for _, enclosure := range item.Enclosures {
				if enclosure != nil {
					enclosuresList = append(enclosuresList, map[string]interface{}{
						"URL":    enclosure.URL,
						"Type":   enclosure.Type,
						"Length": enclosure.Length,
					})
				}
			}
			itemMap["Enclosures"] = enclosuresList
		}

		// Add custom fields if available
		if item.Custom != nil {
			itemMap["Custom"] = item.Custom
		}

		itemsForStorage = append(itemsForStorage, itemMap)
	}

	// Store items in global variable with thread safety
	tempFeedMutex.Lock()
	tempFeedItems = itemsForStorage
	tempFeedMutex.Unlock()

	// Prepare data for template - preserve original feed items for template compatibility
	// Add index to each original item for the template to use
	var itemsWithIndices []interface{} // Use interface{} to hold enhanced gofeed.Item objects
	for i, originalItem := range feed.Items {
		// Create a map that combines the original item with the index
		itemWithIndex := map[string]interface{}{}

		// Copy all fields from the original gofeed.Item
		itemWithIndex["Title"] = originalItem.Title
		itemWithIndex["Description"] = originalItem.Description
		itemWithIndex["Content"] = originalItem.Content
		itemWithIndex["Link"] = originalItem.Link
		itemWithIndex["Updated"] = originalItem.Updated
		itemWithIndex["Published"] = originalItem.Published
		itemWithIndex["GUID"] = originalItem.GUID
		itemWithIndex["Author"] = originalItem.Author
		itemWithIndex["Authors"] = originalItem.Authors
		itemWithIndex["Categories"] = originalItem.Categories
		itemWithIndex["Image"] = originalItem.Image
		itemWithIndex["Links"] = originalItem.Links
		itemWithIndex["UpdatedParsed"] = originalItem.UpdatedParsed
		itemWithIndex["PublishedParsed"] = originalItem.PublishedParsed
		itemWithIndex["Enclosures"] = originalItem.Enclosures
		itemWithIndex["Custom"] = originalItem.Custom

		// Add the index for the form
		itemWithIndex["Index"] = i

		itemsWithIndices = append(itemsWithIndices, itemWithIndex)
	}

	// Prepare data for template
	data := map[string]interface{}{
		"Feed":  feed,
		"Items": itemsWithIndices,
		"URL":   urlStr,
	}

	// Render the index page with the feed data
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
	tmpl.Execute(w, data)
}

// IndexPostHandler handles RSS feed preview and test Telegram submissions
func (h *Handlers) IndexPostHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Check if this is a test Telegram request by index
	itemIndexStr := r.FormValue("item_index")
	if itemIndexStr != "" {
		// Handle test Telegram request by index
		h.handleTestTelegramByIndex(w, r)
		return
	}

	// Handle feed preview request (POST only)
	urlStr := r.FormValue("url")
	if urlStr == "" {
		// Re-render the index page with an error message
		data := map[string]interface{}{
			"Error": "URL is required",
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	// Process the feed preview using shared function
	h.processFeedPreview(w, urlStr)
}

// handleTestTelegramByIndex handles testing Telegram notifications by retrieving the item from global storage using its index
func (h *Handlers) handleTestTelegramByIndex(w http.ResponseWriter, r *http.Request) {
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

	// Create the feed map with feed-level information
	feedMap := map[string]interface{}{
		"Title":       r.FormValue("feed_title"),
		"Description": r.FormValue("feed_description"),
		"Link":        r.FormValue("feed_link"),
		"Language":    r.FormValue("feed_language"),
		"Copyright":   r.FormValue("feed_copyright"),
		"Generator":   r.FormValue("feed_generator"),
		"FeedType":    r.FormValue("feed_type"),
		"FeedVersion": r.FormValue("feed_version"),
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
	message = ProcessFeedItemForTelegram(item, feedMap, message)

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

	// Redirect back to the index page with the feed URL
	if feedUrl != "" {
		http.Redirect(w, r, "/?url="+feedUrl, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
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
	newConfig.Feeds = processFeedsFromForm(r)

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

// processFeedsFromForm processes the feed configuration from the form data
func processFeedsFromForm(r *http.Request) []Feed {
	feedUrls := r.Form["feed_urls"]
	feedIntervals := r.Form["feed_intervals"]
	feedRetentionDays := r.Form["feed_retention_days"]
	telegramTokens := r.Form["telegram_tokens"]
	telegramChatIds := r.Form["telegram_chat_ids"]
	telegramThreadIds := r.Form["telegram_thread_ids"]
	telegramTemplates := r.Form["telegram_templates"]

	var feeds []Feed

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

			feeds = append(feeds, feed)
		}
	}

	return feeds
}

// extractItemFromForm extracts item data from the form request
func extractItemFromForm(r *http.Request) map[string]interface{} {
	title := r.FormValue("title")
	description := r.FormValue("description")
	link := r.FormValue("link")
	content := r.FormValue("content")
	updated := r.FormValue("updated")
	published := r.FormValue("published")
	guid := r.FormValue("guid")
	author := r.FormValue("author")
	authors := r.FormValue("authors")
	categories := r.FormValue("categories")
	imageURL := r.FormValue("image_url")
	imageTitle := r.FormValue("image_title")
	authorEmail := r.FormValue("author_email")
	links := r.FormValue("links")
	updatedParsed := r.FormValue("updated_parsed")
	publishedParsed := r.FormValue("published_parsed")
	enclosures := r.FormValue("enclosures")
	custom := r.FormValue("custom")

	item := map[string]interface{}{
		"Title":       title,
		"Description": description,
		"Link":        link,
		"Content":     content,
		"Updated":     updated,
		"Published":   published,
		"GUID":        guid,
	}

	// Add author information if available
	if author != "" {
		item["Author"] = map[string]interface{}{
			"Name": author,
		}
		if authorEmail != "" {
			item["Author"].(map[string]interface{})["Email"] = authorEmail
		}
	}

	// Add multiple authors if available
	if authors != "" {
		authorList := []interface{}{map[string]interface{}{
			"Name": authors,
		}}
		item["Authors"] = authorList
	}

	// Add categories if available
	if categories != "" {
		item["Categories"] = []interface{}{categories}
	}

	// Add image information if available
	if imageURL != "" {
		imageInfo := map[string]interface{}{
			"URL": imageURL,
		}
		if imageTitle != "" {
			imageInfo["Title"] = imageTitle
		}
		item["Image"] = imageInfo
	}

	// Add other optional fields if available
	if links != "" {
		item["Links"] = []interface{}{links}
	}
	if updatedParsed != "" {
		item["UpdatedParsed"] = updatedParsed
	}
	if publishedParsed != "" {
		item["PublishedParsed"] = publishedParsed
	}
	if enclosures != "" {
		item["Enclosures"] = []interface{}{enclosures}
	}
	if custom != "" {
		item["Custom"] = map[string]interface{}{"info": custom}
	}

	return item
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

	// Create a feed map with feed-level information
	feedMap := map[string]interface{}{
		"Title":       "", // Would need to pass actual feed title
		"Description": "", // Would need to pass actual feed description
		"Link":        feed.FeedUrl,
		"Language":    "", // Would need to pass actual feed language
		"Copyright":   "", // Would need to pass actual feed copyright
		"Generator":   "", // Would need to pass actual feed generator
		"FeedType":    "", // Would need to pass actual feed type
		"FeedVersion": "", // Would need to pass actual feed version
	}

	// Process the feed item for Telegram
	message := ProcessFeedItemForTelegram(item, feedMap, template)

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
