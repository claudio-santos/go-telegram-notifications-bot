package internal

// Config represents the configuration structure
type Config struct {
	Server                      string `yaml:"server"`
	Database                    string `yaml:"database"`
	TestTelegramApiToken        string `yaml:"test_telegram_api_token"`
	TestTelegramChatId          int64  `yaml:"test_telegram_chat_id"`
	TestTelegramMessageThreadId int64  `yaml:"test_telegram_message_thread_id"`
	TestTelegramTemplate        string `yaml:"test_telegram_template"`
	Feeds                       []Feed `yaml:"feeds"`
}

// Feed represents a single RSS feed configuration
type Feed struct {
	FeedUrl                  string `yaml:"feed_url"`
	FeedFetchIntervalMinutes int    `yaml:"feed_fetch_interval_minutes"`
	FeedRetentionDays        int    `yaml:"feed_retention_days"`
	TelegramApiToken         string `yaml:"telegram_api_token"`
	TelegramChatId           int64  `yaml:"telegram_chat_id"`
	TelegramMessageThreadId  int64  `yaml:"telegram_message_thread_id"`
	TelegramTemplate         string `yaml:"telegram_template"`
}

/*
Template Variables Reference (Based on gofeed structures):
The following variables are available for use in Telegram message templates, organized by the gofeed.Item structure:

Basic Feed Item Properties:
- {{.Title}}           : Title of the feed item (from Item.Title)
- {{.Description}}     : Description or summary of the feed item (from Item.Description)
- {{.Content}}         : Full content of the feed item (from Item.Content)
- {{.Link}}            : URL link to the original article (from Item.Link)
- {{.Updated}}         : Update timestamp as string (from Item.Updated)
- {{.Published}}       : Publication timestamp as string (from Item.Published)
- {{.GUID}}            : Globally unique identifier for the item (from Item.GUID)

Author Information (from gofeed.Item.Author and gofeed.Item.Authors):
- {{.Author}}          : Author name (from Item.Author.Name)
- {{.AuthorEmail}}     : Author email address (from Item.Author.Email)
- {{.Authors}}         : All authors with names and emails (from Item.Authors slice)

Category Information (from gofeed.Item.Categories):
- {{.Categories}}      : Comma-separated list of categories (from Item.Categories slice)

Media and Image Information (from gofeed.Item.Image):
- {{.ImageURL}}        : URL of the featured image (from Item.Image.URL)
- {{.ImageTitle}}      : Title/alt text of the featured image (from Item.Image.Title)

Links Information (from gofeed.Item.Links):
- {{.Links}}           : Additional links associated with the item (from Item.Links slice)

Enclosures Information (from gofeed.Item.Enclosures):
- {{.Enclosures}}      : Media enclosures (audio, video, etc.) (from Item.Enclosures slice)
  Each enclosure has: URL, Length, Type (accessed as part of the combined string)

Date and Time Information (from gofeed.Item fields):
- {{.UpdatedParsed}}   : Parsed update timestamp (from Item.UpdatedParsed)
- {{.PublishedParsed}} : Parsed publication timestamp (from Item.PublishedParsed)

Custom Fields (from gofeed.Item.Custom):
- {{.Custom}}          : Any custom fields in the feed (from Item.Custom map)

Feed-Level Properties (from gofeed.Feed):
- {{.FeedLink}}        : URL of the feed itself (from Feed.FeedLink)
- {{.FeedLanguage}}    : Language of the feed (from Feed.Language)
- {{.FeedCopyright}}   : Copyright information (from Feed.Copyright)
- {{.FeedGenerator}}   : Generator of the feed (from Feed.Generator)
- {{.FeedType}}        : Type of the feed (RSS, Atom, etc.) (from Feed.FeedType)
- {{.FeedVersion}}     : Version of the feed format (from Feed.FeedVersion)

All possible template variables supported by the system:
- {{.Title}}
- {{.Description}}
- {{.Content}}
- {{.Link}}
- {{.Links}}
- {{.Updated}}
- {{.UpdatedParsed}}
- {{.Published}}
- {{.PublishedParsed}}
- {{.Author}}
- {{.AuthorEmail}}
- {{.Authors}}
- {{.GUID}}
- {{.ImageURL}}
- {{.ImageTitle}}
- {{.Categories}}
- {{.Enclosures}}
- {{.Custom}}

Note: Some feed-level variables like FeedLink, FeedLanguage, etc. are not currently accessible from individual items
but are available when processing the entire feed. The current implementation focuses on item-level variables.

These variables correspond to the gofeed.Item structure fields and can be used in both test templates and feed-specific templates.
*/
// TelegramMessage represents the structure for sending messages to Telegram
type TelegramMessage struct {
	ChatID              int64  `json:"chat_id"`
	Text                string `json:"text"`
	ParseMode           string `json:"parse_mode,omitempty"`
	MessageThreadID     int64  `json:"message_thread_id,omitempty"`
	DisableNotification bool   `json:"disable_notification,omitempty"`
}
