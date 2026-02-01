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
