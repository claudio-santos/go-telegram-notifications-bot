# Go Telegram Notifications Bot

A Go-based application that monitors RSS feeds and sends notifications to Telegram when new items are published. The application provides a web interface for easy configuration and testing.

## Features

- Monitor multiple RSS feeds at configurable intervals
- Send feed updates to Telegram chats using bot API tokens
- Customizable message templates with various feed item variables
- Web interface for configuration and testing
- XSS protection through HTML sanitization
- Support for Telegram threads (topics)
- Preview RSS feeds directly in the web interface
- Automatic cleanup of old feed items based on retention settings

## Prerequisites

- Go 1.25.6 or higher
- A Telegram bot token (get one from [@BotFather](https://t.me/BotFather))

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd go-telegram-notifications-bot
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Copy the sample configuration file:
   ```bash
   cp config.yaml.dummy config.yaml
   ```

4. Edit `config.yaml` with your settings (see Configuration section below)

5. Build the application:
   ```bash
   go build -o go-telegram-notifications-bot
   ```

## Configuration

The application uses a `config.yaml` file for configuration. Here's the structure:

```yaml
server: "8080"  # Port for the web server
database: database.db  # Path for the SQLite database file
test_telegram_api_token: <YOUR_BOT_API_TOKEN>  # Telegram bot API token for testing
test_telegram_chat_id: <YOUR_CHAT_ID>  # Target chat ID for testing
test_telegram_message_thread_id: <THREAD_ID>  # Message thread ID for testing (optional)
test_telegram_template: "<b><a href=\"{{.Link}}\">{{.Title}}</a></b>\r\n{{.Description}}"  # Template for test messages
feeds:
    - feed_url: <RSS_FEED_URL>  # URL of the RSS feed
      feed_fetch_interval_minutes: 60  # How often to check for updates (in minutes)
      feed_retention_days: 30  # How many days to keep feed items
      telegram_api_token: <YOUR_BOT_API_TOKEN>  # Telegram bot API token
      telegram_chat_id: <YOUR_CHAT_ID>  # Target chat ID
      telegram_message_thread_id: <THREAD_ID>  # Message thread ID (optional)
      telegram_template: '{{.Description}}'  # Template for Telegram messages
```

### Template Variables

You can use the following variables in your message templates:

#### Item Variables:
- `{{.Title}}` - Title of the feed item
- `{{.Description}}` - Description or summary of the feed item
- `{{.Content}}` - Full content of the feed item
- `{{.Link}}` - URL link to the original article
- `{{.Links}}` - Additional links associated with the item
- `{{.Updated}}` - Update timestamp as string
- `{{.UpdatedParsed}}` - Parsed update timestamp
- `{{.Published}}` - Publication timestamp as string
- `{{.PublishedParsed}}` - Parsed publication timestamp
- `{{.Author}}` - Author name
- `{{.AuthorEmail}}` - Author email address
- `{{.Authors}}` - All authors with names and emails
- `{{.GUID}}` - Globally unique identifier for the item
- `{{.ImageURL}}` - URL of the featured image
- `{{.ImageTitle}}` - Title/alt text of the featured image
- `{{.Categories}}` - Comma-separated list of categories
- `{{.Enclosures}}` - Media enclosures (audio, video, etc.)
- `{{.Custom}}` - Any custom fields in the feed

#### Feed Variables:
- `{{.FeedTitle}}` - Title of the feed itself
- `{{.FeedDescription}}` - Description of the feed
- `{{.FeedLink}}` - URL of the feed itself
- `{{.FeedLanguage}}` - Language of the feed
- `{{.FeedCopyright}}` - Copyright information
- `{{.FeedGenerator}}` - Generator of the feed
- `{{.FeedType}}` - Type of the feed (RSS, Atom, etc.)
- `{{.FeedVersion}}` - Version of the feed format

## Usage

1. Start the application:
   ```bash
   ./go-telegram-notifications-bot
   ```

2. Open your browser and navigate to `http://localhost:8080`

3. Use the web interface to:
   - Preview RSS feeds
   - Configure your feeds and Telegram settings
   - Test Telegram notifications

## Web Interface

The application provides a web interface with two main pages:

### RSS Preview (`/`)
- Enter an RSS feed URL to preview its content
- See detailed information about the feed and its items
- Test sending individual feed items to Telegram

### Configuration (`/config`)
- Configure server settings
- Set up Telegram API tokens and chat IDs
- Add and configure multiple RSS feeds
- Customize message templates

## Security

The application includes security measures to prevent XSS attacks by sanitizing HTML content before displaying it or sending it to Telegram. Only a safe subset of HTML tags is allowed in messages.

## Dependencies

This project uses the following Go packages:
- [chi](https://github.com/go-chi/chi) - Lightweight, idiomatic HTTP router
- [gofeed](https://github.com/mmcdole/gofeed) - RSS/Atom feed parser
- [bluemonday](https://github.com/microcosm-cc/bluemonday) - HTML sanitizer
- [yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3) - YAML parser

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the GNU General Public License v3.0 (GPLv3) - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Go](https://golang.org/)
- Uses [chi](https://github.com/go-chi/chi) router
- RSS parsing with [gofeed](https://github.com/mmcdole/gofeed)
- HTML sanitization with [bluemonday](https://github.com/microcosm-cc/bluemonday)
- Frontend styling with [Tabler](https://tabler.io/)