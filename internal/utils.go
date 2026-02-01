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
	// Replace template variables with actual values from the gofeed item
	titleStr, _ := item["Title"].(string)
	descriptionStr, _ := item["Description"].(string)
	contentStr, _ := item["Content"].(string)
	linkStr, _ := item["Link"].(string)
	updatedStr, _ := item["Updated"].(string)
	publishedStr, _ := item["Published"].(string)
	guidStr, _ := item["GUID"].(string)

	// Get feed-level information
	feedTitle, _ := feed["Title"].(string)
	feedDescription, _ := feed["Description"].(string)
	feedLink, _ := feed["Link"].(string)
	feedLanguage, _ := feed["Language"].(string)
	feedCopyright, _ := feed["Copyright"].(string)
	feedGenerator, _ := feed["Generator"].(string)
	feedType, _ := feed["FeedType"].(string)
	feedVersion, _ := feed["FeedVersion"].(string)

	// Get author information
	authorInterface := item["Author"]
	var authorNameStr, authorEmailStr string
	if authorInterface != nil {
		// Handle different possible author structures
		switch v := authorInterface.(type) {
		case map[string]interface{}:
			// Standard author object with Name and Email fields
			if name, ok := v["Name"].(string); ok {
				authorNameStr = name
			} else if name, ok := v["name"].(string); ok {
				authorNameStr = name
			} else if name, ok := v["Name"]; ok {
				// Handle if Name is not a string
				authorNameStr = fmt.Sprintf("%v", name)
			}

			if email, ok := v["Email"].(string); ok {
				authorEmailStr = email
			} else if email, ok := v["email"].(string); ok {
				authorEmailStr = email
			} else if email, ok := v["Email"]; ok {
				// Handle if Email is not a string
				authorEmailStr = fmt.Sprintf("%v", email)
			}
		case string:
			// Sometimes author is just a string
			authorNameStr = v
		case map[string]string:
			// Handle if it's a string map
			if name, ok := v["Name"]; ok {
				authorNameStr = name
			} else if name, ok := v["name"]; ok {
				authorNameStr = name
			}

			if email, ok := v["Email"]; ok {
				authorEmailStr = email
			} else if email, ok := v["email"]; ok {
				authorEmailStr = email
			}
		default:
			// Handle any other type by converting to string
			authorNameStr = fmt.Sprintf("%v", authorInterface)
		}
	}

	// Get multiple authors
	authorsInterface := item["Authors"]
	var allAuthorsStr string
	if authorsInterface != nil {
		switch v := authorsInterface.(type) {
		case []interface{}:
			var authors []string
			for _, author := range v {
				switch authorType := author.(type) {
				case map[string]interface{}:
					var authorParts []string
					if name, ok := authorType["Name"].(string); ok {
						authorParts = append(authorParts, name)
					} else if name, ok := authorType["name"].(string); ok {
						authorParts = append(authorParts, name)
					} else if name, ok := authorType["Name"]; ok {
						authorParts = append(authorParts, fmt.Sprintf("%v", name))
					}

					if email, ok := authorType["Email"].(string); ok {
						authorParts = append(authorParts, "<"+email+">")
					} else if email, ok := authorType["email"].(string); ok {
						authorParts = append(authorParts, "<"+email+">")
					} else if email, ok := authorType["Email"]; ok {
						authorParts = append(authorParts, "<"+fmt.Sprintf("%v", email)+">")
					}

					if len(authorParts) > 0 {
						authors = append(authors, strings.Join(authorParts, " "))
					}
				case string:
					authors = append(authors, authorType)
				default:
					authors = append(authors, fmt.Sprintf("%v", author))
				}
			}
			allAuthorsStr = strings.Join(authors, "; ")
		case string:
			allAuthorsStr = v
		default:
			allAuthorsStr = fmt.Sprintf("%v", authorsInterface)
		}
	}

	// Get categories
	categoriesInterface := item["Categories"]
	var categoriesStr string
	if categoriesInterface != nil {
		switch v := categoriesInterface.(type) {
		case []interface{}:
			var cats []string
			for _, cat := range v {
				if catStr, ok := cat.(string); ok {
					cats = append(cats, catStr)
				} else {
					cats = append(cats, fmt.Sprintf("%v", cat))
				}
			}
			categoriesStr = strings.Join(cats, ", ")
		case []string:
			categoriesStr = strings.Join(v, ", ")
		case string:
			categoriesStr = v
		default:
			categoriesStr = fmt.Sprintf("%v", categoriesInterface)
		}
	}

	// Get links
	linksInterface := item["Links"]
	var linksStr string
	if linksInterface != nil {
		linksSlice, ok := linksInterface.([]interface{})
		if ok {
			var links []string
			for _, link := range linksSlice {
				if linkStr, ok := link.(string); ok {
					links = append(links, linkStr)
				}
			}
			linksStr = strings.Join(links, ", ")
		}
	}

	// Get enclosures
	enclosuresInterface := item["Enclosures"]
	var enclosuresStr string
	if enclosuresInterface != nil {
		enclosuresSlice, ok := enclosuresInterface.([]interface{})
		if ok {
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
			enclosuresStr = strings.Join(enclosures, "; ")
		}
	}

	// Get image information
	imageInterface := item["Image"]
	var imageURLStr, imageTitleStr string
	if imageInterface != nil {
		imageMap, ok := imageInterface.(map[string]interface{})
		if ok {
			if url, ok := imageMap["URL"].(string); ok {
				imageURLStr = url
			} else if url, ok := imageMap["url"].(string); ok {
				imageURLStr = url
			} else if url, ok := imageMap["URL"]; ok {
				imageURLStr = fmt.Sprintf("%v", url)
			}

			if title, ok := imageMap["Title"].(string); ok {
				imageTitleStr = title
			} else if title, ok := imageMap["title"].(string); ok {
				imageTitleStr = title
			} else if title, ok := imageMap["Title"]; ok {
				imageTitleStr = fmt.Sprintf("%v", title)
			}
		} else {
			// Handle if imageInterface is a string (direct URL)
			if str, ok := imageInterface.(string); ok {
				imageURLStr = str
			} else {
				imageURLStr = fmt.Sprintf("%v", imageInterface)
			}
		}
	}

	// Get custom fields
	customInterface := item["Custom"]
	var customStr string
	if customInterface != nil {
		customMap, ok := customInterface.(map[string]interface{})
		if ok {
			var customs []string
			for key, value := range customMap {
				if valueStr, ok := value.(string); ok {
					customs = append(customs, key+": "+valueStr)
				}
			}
			customStr = strings.Join(customs, "; ")
		}
	}

	// Get UpdatedParsed and PublishedParsed
	updatedParsedInterface := item["UpdatedParsed"]
	var updatedParsedStr string
	if updatedParsedInterface != nil {
		updatedParsedStr = fmt.Sprintf("%v", updatedParsedInterface)
	}

	publishedParsedInterface := item["PublishedParsed"]
	var publishedParsedStr string
	if publishedParsedInterface != nil {
		publishedParsedStr = fmt.Sprintf("%v", publishedParsedInterface)
	}

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

// ReplaceTemplateVars replaces template variables with actual values
func ReplaceTemplateVars(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}
