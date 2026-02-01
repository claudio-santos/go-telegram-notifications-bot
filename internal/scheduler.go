package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

// FeedScheduler manages scheduling and fetching of feeds
type FeedScheduler struct {
	configManager *ConfigManager
	dbManager     *DBManager
	telegram      *TelegramService
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.Mutex
	tickers       map[string]*time.Ticker
}

// NewFeedScheduler creates a new feed scheduler
func NewFeedScheduler(cm *ConfigManager, dbm *DBManager) *FeedScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &FeedScheduler{
		configManager: cm,
		dbManager:     dbm,
		telegram:      NewTelegramService(cm),
		ctx:           ctx,
		cancel:        cancel,
		tickers:       make(map[string]*time.Ticker),
	}
}

// Start begins the feed scheduling process
func (fs *FeedScheduler) Start() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Stop any existing tickers
	for url, ticker := range fs.tickers {
		ticker.Stop()
		delete(fs.tickers, url)
	}

	// Perform initial fetch for each feed
	for _, feed := range fs.configManager.Config.Feeds {
		log.Printf("Performing initial fetch for feed: %s", feed.FeedUrl)
		err := fs.fetchAndProcessFeed(feed)
		if err != nil {
			log.Printf("Error during initial fetch for feed %s: %v", feed.FeedUrl, err)
		}
	}

	// Start new tickers for each feed
	for _, feed := range fs.configManager.Config.Feeds {
		fs.startTickerForFeed(feed)
	}

	log.Println("Feed scheduler started")
}

// startTickerForFeed starts a ticker for a specific feed
func (fs *FeedScheduler) startTickerForFeed(feed Feed) {
	// Stop existing ticker if present
	if existingTicker, exists := fs.tickers[feed.FeedUrl]; exists {
		existingTicker.Stop()
	}

	interval := time.Duration(feed.FeedFetchIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)

	fs.tickers[feed.FeedUrl] = ticker

	// Start goroutine to handle ticker ticks
	fs.wg.Add(1)
	go func(f Feed) {
		defer fs.wg.Done()
		for {
			select {
			case <-ticker.C:
				err := fs.fetchAndProcessFeed(f)
				if err != nil {
					log.Printf("Error processing feed %s: %v", f.FeedUrl, err)
				}
			case <-fs.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}(feed)

	log.Printf("Started scheduler for feed: %s (interval: %d minutes)", feed.FeedUrl, feed.FeedFetchIntervalMinutes)
}

// fetchAndProcessFeed fetches a feed and processes its items
func (fs *FeedScheduler) fetchAndProcessFeed(feed Feed) error {
	log.Printf("Fetching feed: %s", feed.FeedUrl)

	fp := gofeed.NewParser()
	feedData, err := fp.ParseURL(feed.FeedUrl)
	if err != nil {
		return fmt.Errorf("failed to parse feed %s: %v", feed.FeedUrl, err)
	}

	// Process items in reverse order (oldest first) to maintain chronological order
	for i := len(feedData.Items) - 1; i >= 0; i-- {
		item := feedData.Items[i]

		// Check if this item has already been posted
		isPosted, err := fs.dbManager.IsFeedItemPosted(item.GUID, feed.FeedUrl)
		if err != nil {
			log.Printf("Error checking if item is posted: %v", err)
			continue
		}

		if isPosted {
			continue // Skip already posted items
		}

		// Convert gofeed.Item to our FeedItem struct
		feedItem := FeedItem{
			GUID:        item.GUID,
			Title:       item.Title,
			Description: item.Description,
			Link:        item.Link,
			FeedURL:     feed.FeedUrl,
		}

		if item.PublishedParsed != nil {
			feedItem.PublishedAt = *item.PublishedParsed
		} else {
			feedItem.PublishedAt = time.Now()
		}

		// Create itemMap for Telegram
		itemMap := map[string]interface{}{
			"Title":       item.Title,
			"Description": item.Description,
			"Content":     item.Content,
			"Link":        item.Link,
			"Updated":     item.Updated,
			"Published":   item.Published,
			"GUID":        item.GUID,

			"Author": func() interface{} {
				if item.Author != nil {
					return map[string]interface{}{
						"Name":  item.Author.Name,
						"Email": item.Author.Email,
					}
				}
				return nil
			}(),

			"Authors": func() []interface{} {
				var authorsList []interface{}
				for _, author := range item.Authors {
					if author != nil {
						authorsList = append(authorsList, map[string]interface{}{
							"Name":  author.Name,
							"Email": author.Email,
						})
					}
				}
				return authorsList
			}(),

			// Categories
			"Categories": item.Categories,

			// Image information
			"Image": func() interface{} {
				if item.Image != nil {
					return map[string]interface{}{
						"URL":   item.Image.URL,
						"Title": item.Image.Title,
					}
				}
				return nil
			}(),

			// Links
			"Links": item.Links,

			// Date/time information
			"UpdatedParsed": func() string {
				if item.UpdatedParsed != nil {
					return item.UpdatedParsed.Format("2006-01-02 15:04:05 MST")
				}
				return ""
			}(),
			"PublishedParsed": func() string {
				if item.PublishedParsed != nil {
					return item.PublishedParsed.Format("2006-01-02 15:04:05 MST")
				}
				return ""
			}(),

			// Enclosures
			"Enclosures": func() []interface{} {
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
				return enclosuresList
			}(),

			// Custom fields
			"Custom": item.Custom,

			// Feed-level properties
			"FeedTitle":       feedData.Title,
			"FeedDescription": feedData.Description,
			"FeedLink":        feedData.Link,
			"FeedLanguage":    feedData.Language,
			"FeedCopyright":   feedData.Copyright,
			"FeedGenerator":   feedData.Generator,
			"FeedType":        feedData.FeedType,
			"FeedVersion":     feedData.FeedVersion,
		}

		// Send the item to Telegram first
		err = fs.telegram.SendFeedItemToTelegram(feed, itemMap)
		if err != nil {
			log.Printf("Error sending feed item to Telegram: %v", err)
			// Don't save to database if sending to Telegram failed
			continue
		}

		// Save the item to the database after successful send
		err = fs.dbManager.SaveFeedItem(feedItem)
		if err != nil {
			log.Printf("Error saving feed item: %v", err)
			continue
		} else {
			log.Printf("Sent feed item to Telegram and saved to database: %s", item.Title)
		}
	}

	return nil
}

// Stop stops the feed scheduler
func (fs *FeedScheduler) Stop() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.cancel()

	// Stop all tickers
	for url, ticker := range fs.tickers {
		ticker.Stop()
		delete(fs.tickers, url)
	}

	// Wait for all goroutines to finish
	fs.wg.Wait()

	log.Println("Feed scheduler stopped")
}

// RefreshConfiguration updates the scheduler with new configuration
func (fs *FeedScheduler) RefreshConfiguration() {
	fs.Start() // Restart with new configuration
}

// StartCleanupRoutine starts a periodic cleanup routine
func (fs *FeedScheduler) StartCleanupRoutine() {
	fs.wg.Add(1)
	go func() {
		defer fs.wg.Done()

		// Run cleanup immediately when starting
		fs.runCleanup()

		// Then run every 24 hours
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fs.runCleanup()
			case <-fs.ctx.Done():
				return
			}
		}
	}()

	log.Println("Cleanup routine started")
}

// runCleanup performs the cleanup of old feed items
func (fs *FeedScheduler) runCleanup() {
	log.Println("Starting cleanup of old feed items...")

	for _, feed := range fs.configManager.Config.Feeds {
		if feed.FeedRetentionDays > 0 {
			err := fs.dbManager.CleanupOldItems(feed.FeedRetentionDays)
			if err != nil {
				log.Printf("Error cleaning up old items for feed %s: %v", feed.FeedUrl, err)
			}
		}
	}

	log.Println("Finished cleanup of old feed items")
}
