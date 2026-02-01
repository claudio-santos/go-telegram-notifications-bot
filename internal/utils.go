package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// SendTelegramMessage sends a message to Telegram using the official API
func SendTelegramMessage(token string, msg TelegramMessage) error {
	// Truncate message if it's too long (Telegram has a 4096 character limit)
	const maxMessageLength = 4096
	if len(msg.Text) > maxMessageLength {
		// Try to truncate at a sentence boundary if possible
		truncated := msg.Text[:maxMessageLength]
		lastSentence := strings.LastIndex(truncated, ". ")
		if lastSentence > maxMessageLength/2 { // Only truncate at sentence if it's not too early
			msg.Text = truncated[:lastSentence+1] + "..."
		} else {
			// Otherwise just truncate at the limit with ellipsis
			msg.Text = truncated + "..."
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Send to Telegram API
	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	response, err := http.Post(telegramURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending to Telegram: %v", err)
	}
	defer response.Body.Close()

	// Check if the request was successful
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned error: %s", response.Status)
	}

	// Decode the response to check for API errors
	var apiResponse struct {
		Ok          bool        `json:"ok"`
		Result      interface{} `json:"result"`
		Description string      `json:"description"`
		ErrorCode   int         `json:"error_code"`
	}

	if err := json.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("error decoding Telegram API response: %v", err)
	}

	if !apiResponse.Ok {
		return fmt.Errorf("Telegram API error: %s (code: %d)", apiResponse.Description, apiResponse.ErrorCode)
	}

	return nil
}

// SanitizeText sanitizes input text to allow only a safe subset of HTML tags
func SanitizeText(text string) string {
	policy := bluemonday.StrictPolicy()
	policy.AllowElements("b", "strong", "i", "em", "u", "ins",
		"s", "strike", "del", "code", "pre", "blockquote")
	policy.AllowAttrs("href").OnElements("a")
	sanitized := policy.Sanitize(text)
	return sanitized
}

// ProcessFeedItemForTelegram processes a feed item and feed metadata and prepares it for Telegram messaging
func ProcessFeedItemForTelegram(item map[string]interface{}, feed map[string]interface{}, template string) string {
	// Extract basic item fields
	titleStr := getStringValue(item, "Title")
	descriptionStr := getStringValue(item, "Description")
	contentStr := getStringValue(item, "Content")
	linkStr := getStringValue(item, "Link")
	updatedStr := getStringValue(item, "Updated")
	publishedStr := getStringValue(item, "Published")
	guidStr := getStringValue(item, "GUID")

	// Extract feed-level information
	feedTitle := getStringValue(feed, "Title")
	feedDescription := getStringValue(feed, "Description")
	feedLink := getStringValue(feed, "Link")
	feedLanguage := getStringValue(feed, "Language")
	feedCopyright := getStringValue(feed, "Copyright")
	feedGenerator := getStringValue(feed, "Generator")
	feedType := getStringValue(feed, "FeedType")
	feedVersion := getStringValue(feed, "FeedVersion")

	// Extract complex fields
	authorNameStr, authorEmailStr := extractAuthorInfo(item)
	allAuthorsStr := extractStringList(item, "Authors", "; ")
	categoriesStr := extractStringList(item, "Categories", ", ")
	linksStr := extractStringList(item, "Links", ", ")
	enclosuresStr := extractEnclosures(item)
	imageURLStr, imageTitleStr := extractImageInfo(item)
	customStr := extractCustomFields(item)
	updatedParsedStr := getStringValue(item, "UpdatedParsed")
	publishedParsedStr := getStringValue(item, "PublishedParsed")

	// Sanitize and escape text for Telegram
	titleStr = SanitizeText(titleStr)
	descriptionStr = SanitizeText(descriptionStr)
	contentStr = SanitizeText(contentStr)
	linkStr = SanitizeText(linkStr)
	linksStr = SanitizeText(linksStr)
	updatedStr = SanitizeText(updatedStr)
	updatedParsedStr = SanitizeText(updatedParsedStr)
	publishedStr = SanitizeText(publishedStr)
	publishedParsedStr = SanitizeText(publishedParsedStr)
	authorNameStr = SanitizeText(authorNameStr)
	authorEmailStr = SanitizeText(authorEmailStr)
	allAuthorsStr = SanitizeText(allAuthorsStr)
	guidStr = SanitizeText(guidStr)
	imageURLStr = SanitizeText(imageURLStr)
	imageTitleStr = SanitizeText(imageTitleStr)
	categoriesStr = SanitizeText(categoriesStr)
	enclosuresStr = SanitizeText(enclosuresStr)
	customStr = SanitizeText(customStr)

	// Replace template variables
	message := ReplaceTemplateVars(template, map[string]string{
		".Title":           titleStr,
		".Description":     descriptionStr,
		".Content":         contentStr,
		".Link":            linkStr,
		".Links":           linksStr,
		".Updated":         updatedStr,
		".UpdatedParsed":   updatedParsedStr,
		".Published":       publishedStr,
		".PublishedParsed": publishedParsedStr,
		".Author":          authorNameStr,
		".AuthorEmail":     authorEmailStr,
		".Authors":         allAuthorsStr,
		".GUID":            guidStr,
		".ImageURL":        imageURLStr,
		".ImageTitle":      imageTitleStr,
		".Categories":      categoriesStr,
		".Enclosures":      enclosuresStr,
		".Custom":          customStr,
		".FeedTitle":       feedTitle,
		".FeedDescription": feedDescription,
		".FeedLink":        feedLink,
		".FeedLanguage":    feedLanguage,
		".FeedCopyright":   feedCopyright,
		".FeedGenerator":   feedGenerator,
		".FeedType":        feedType,
		".FeedVersion":     feedVersion,
	})

	return message
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// extractAuthorInfo extracts author information from the item
func extractAuthorInfo(item map[string]interface{}) (name, email string) {
	authorInterface := item["Author"]
	if authorInterface == nil {
		return "", ""
	}

	switch v := authorInterface.(type) {
	case map[string]interface{}:
		if nameVal, ok := v["Name"].(string); ok {
			name = nameVal
		} else if nameVal, ok := v["name"].(string); ok {
			name = nameVal
		} else if nameVal, ok := v["Name"]; ok {
			name = fmt.Sprintf("%v", nameVal)
		}

		if emailVal, ok := v["Email"].(string); ok {
			email = emailVal
		} else if emailVal, ok := v["email"].(string); ok {
			email = emailVal
		} else if emailVal, ok := v["Email"]; ok {
			email = fmt.Sprintf("%v", emailVal)
		}
	case string:
		name = v
	case map[string]string:
		if nameVal, ok := v["Name"]; ok {
			name = nameVal
		} else if nameVal, ok := v["name"]; ok {
			name = nameVal
		}

		if emailVal, ok := v["Email"]; ok {
			email = emailVal
		} else if emailVal, ok := v["email"]; ok {
			email = emailVal
		}
	default:
		name = fmt.Sprintf("%v", authorInterface)
	}

	return name, email
}

// extractStringList extracts a list of strings from an interface and joins them with a separator
func extractStringList(item map[string]interface{}, key, separator string) string {
	value := item[key]
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case []interface{}:
		var items []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				items = append(items, str)
			} else {
				items = append(items, fmt.Sprintf("%v", item))
			}
		}
		return strings.Join(items, separator)
	case []string:
		return strings.Join(v, separator)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", value)
	}
}

// extractEnclosures extracts enclosure information from the item
func extractEnclosures(item map[string]interface{}) string {
	enclosuresInterface := item["Enclosures"]
	if enclosuresInterface == nil {
		return ""
	}

	enclosuresSlice, ok := enclosuresInterface.([]interface{})
	if !ok {
		return fmt.Sprintf("%v", enclosuresInterface)
	}

	var enclosures []string
	for _, enclosure := range enclosuresSlice {
		if enclosureMap, ok := enclosure.(map[string]interface{}); ok {
			var enclosureParts []string
			if url, ok := enclosureMap["URL"].(string); ok {
				enclosureParts = append(enclosureParts, url)
			}
			if typ, ok := enclosureMap["Type"].(string); ok {
				enclosureParts = append(enclosureParts, typ)
			}
			if length, ok := enclosureMap["Length"].(string); ok {
				enclosureParts = append(enclosureParts, length)
			} else if length, ok := enclosureMap["Length"].(float64); ok {
				enclosureParts = append(enclosureParts, fmt.Sprintf("%.0f", length))
			}
			if len(enclosureParts) > 0 {
				enclosures = append(enclosures, strings.Join(enclosureParts, " | "))
			}
		}
	}
	return strings.Join(enclosures, "; ")
}

// extractImageInfo extracts image information from the item
func extractImageInfo(item map[string]interface{}) (url, title string) {
	imageInterface := item["Image"]
	if imageInterface == nil {
		return "", ""
	}

	imageMap, ok := imageInterface.(map[string]interface{})
	if ok {
		if urlVal, ok := imageMap["URL"].(string); ok {
			url = urlVal
		} else if urlVal, ok := imageMap["url"].(string); ok {
			url = urlVal
		} else if urlVal, ok := imageMap["URL"]; ok {
			url = fmt.Sprintf("%v", urlVal)
		}

		if titleVal, ok := imageMap["Title"].(string); ok {
			title = titleVal
		} else if titleVal, ok := imageMap["title"].(string); ok {
			title = titleVal
		} else if titleVal, ok := imageMap["Title"]; ok {
			title = fmt.Sprintf("%v", titleVal)
		}
	} else {
		// Handle if imageInterface is a string (direct URL)
		if str, ok := imageInterface.(string); ok {
			url = str
		} else {
			url = fmt.Sprintf("%v", imageInterface)
		}
	}

	return url, title
}

// extractCustomFields extracts custom fields from the item
func extractCustomFields(item map[string]interface{}) string {
	customInterface := item["Custom"]
	if customInterface == nil {
		return ""
	}

	customMap, ok := customInterface.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", customInterface)
	}

	var customs []string
	for key, value := range customMap {
		if valueStr, ok := value.(string); ok {
			customs = append(customs, key+": "+valueStr)
		} else {
			customs = append(customs, key+": "+fmt.Sprintf("%v", value))
		}
	}
	return strings.Join(customs, "; ")
}

// ReplaceTemplateVars replaces template variables with actual values
func ReplaceTemplateVars(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
