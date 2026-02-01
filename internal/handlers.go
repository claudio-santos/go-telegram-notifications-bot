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

// Global variables for temporary storage with thread safety
var (
	tempFeedItems []map[string]interface{}
	tempFeedInfo  *gofeed.Feed
	tempFeedMutex sync.RWMutex
)

// Handlers manages all HTTP handlers
type Handlers struct {
	ConfigManager   *ConfigManager
	TelegramService *TelegramService
	Scheduler       *FeedScheduler
}

// NewHandlers creates a new Handlers instance
func NewHandlers(cm *ConfigManager, scheduler *FeedScheduler) *Handlers {
	return &Handlers{
		ConfigManager:   cm,
		TelegramService: NewTelegramService(cm),
		Scheduler:       scheduler,
	}
}

// IndexGetHandler serves the home page.
func (h *Handlers) IndexGetHandler(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.Query().Get("url")
	if urlStr != "" {
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

	// Store items and feed info in global variable with thread safety
	tempFeedMutex.Lock()
	tempFeedItems = itemsForStorage
	tempFeedInfo = feed
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

// IndexPostHandler handles RSS feed preview and test Telegram submissions.
func (h *Handlers) IndexPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	itemIndexStr := r.FormValue("item_index")
	if itemIndexStr != "" {
		h.TelegramService.HandleTestTelegramByIndex(w, r)
		return
	}

	urlStr := r.FormValue("url")
	if urlStr == "" {
		data := map[string]interface{}{
			"Error": "URL is required",
			"URL":   urlStr,
		}
		tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/partials/navbar.html"))
		tmpl.Execute(w, data)
		return
	}

	h.processFeedPreview(w, urlStr)
}

// ConfigGetHandler serves the configuration page.
func (h *Handlers) ConfigGetHandler(w http.ResponseWriter, r *http.Request) {
	addEmptyFeed := r.URL.Query().Get("add_feed") == "true"

	feeds := h.ConfigManager.Config.Feeds
	if addEmptyFeed {
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

// ConfigPostHandler updates the configuration from form data.
func (h *Handlers) ConfigPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
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

	newConfig := Config{
		Server:                      r.FormValue("server"),
		Database:                    r.FormValue("database"),
		TestTelegramApiToken:        r.FormValue("test_telegram_api_token"),
		TestTelegramChatId:          0,
		TestTelegramMessageThreadId: 0,
		TestTelegramTemplate:        r.FormValue("test_telegram_template"),
		Feeds:                       []Feed{},
	}

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

	newConfig.Feeds = processFeedsFromForm(r)

	h.ConfigManager.Config = &newConfig

	err = h.ConfigManager.SaveConfig()
	if err != nil {
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

	// Refresh the scheduler with the new configuration
	if h.Scheduler != nil {
		h.Scheduler.RefreshConfiguration()
	}

	http.Redirect(w, r, "/config", http.StatusSeeOther)
}

// processFeedsFromForm processes the feed configuration from the form data.
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
		if feedUrls[i] != "" {
			interval := 30
			if i < len(feedIntervals) && feedIntervals[i] != "" {
				if val, err := strconv.Atoi(feedIntervals[i]); err == nil {
					interval = val
				}
			}

			retentionDays := 30
			if i < len(feedRetentionDays) && feedRetentionDays[i] != "" {
				if val, err := strconv.Atoi(feedRetentionDays[i]); err == nil {
					retentionDays = val
				}
			}

			chatId := int64(0)
			if i < len(telegramChatIds) && telegramChatIds[i] != "" {
				if val, err := strconv.ParseInt(telegramChatIds[i], 10, 64); err == nil {
					chatId = val
				}
			}

			threadId := int64(0)
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
