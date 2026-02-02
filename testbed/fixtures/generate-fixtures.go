// Package main generates deterministic fixture data for E2E testing.
// Usage: go run generate-fixtures.go <collection> <count> > output.jsonl
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// Seeded RNG for reproducibility
var rng = rand.New(rand.NewSource(42))

// Unicode test strings for diverse character coverage
var unicodeNames = []string{
	// CJK characters
	"李明", "王芳", "张伟", "田中太郎", "鈴木一郎", "김철수", "박영희",
	// Cyrillic
	"Иван Петров", "Мария Сидорова", "Олександр Коваленко",
	// Arabic
	"محمد أحمد", "فاطمة علي", "عبدالله محمود",
	// Hebrew
	"דוד כהן", "שרה לוי",
	// Greek
	"Νίκος Παπαδόπουλος", "Μαρία Γεωργίου",
	// Devanagari (Hindi)
	"राहुल शर्मा", "प्रिया गुप्ता",
	// Thai
	"สมชาย ใจดี", "สุนิสา รักษ์ไทย",
	// Emoji in names
	"Alex 🚀", "Sam ⭐", "Jordan 🎸",
	// Mixed scripts
	"Jean-Pierre Müller", "José García López", "Björk Guðmundsdóttir",
	// Combining characters
	"Café Owner", "naïve résumé",
	// RTL with numbers
	"محمد 123", "שלום 456",
}

var brands = []string{
	"Apple", "Samsung", "Sony", "LG", "Microsoft", "Google", "Amazon",
	"Dell", "HP", "Lenovo", "Asus", "Acer", "Nike", "Adidas", "Puma",
}

var categories = []string{
	"Electronics", "Clothing", "Home & Garden", "Sports", "Books",
	"Toys", "Automotive", "Health", "Beauty", "Food & Beverages",
}

var articleCategories = []string{
	"Technology", "Science", "Health", "Business", "Entertainment",
	"Sports", "Politics", "Education", "Travel", "Lifestyle",
}

var eventTypes = []string{
	"Conference", "Workshop", "Meetup", "Webinar", "Concert",
	"Festival", "Exhibition", "Seminar", "Hackathon", "Networking",
}

var timezones = []string{
	"America/New_York", "America/Los_Angeles", "Europe/London",
	"Europe/Paris", "Asia/Tokyo", "Asia/Shanghai", "Australia/Sydney",
	"Pacific/Auckland", "America/Sao_Paulo", "Africa/Cairo",
}

var locales = []string{
	"en-US", "en-GB", "zh-CN", "zh-TW", "ja-JP", "ko-KR",
	"ar-SA", "he-IL", "ru-RU", "de-DE", "fr-FR", "es-ES",
	"pt-BR", "hi-IN", "th-TH",
}

// Special characters for edge case testing
var specialCharsStrings = []string{
	"Line1\nLine2\nLine3",
	"Tab\there\tand\tthere",
	`Backslash: \ and quote: "`,
	"<script>alert('xss')</script>",
	"SELECT * FROM users; DROP TABLE users;--",
	`{"nested": "json", "value": 123}`,
	"https://example.com/path?query=value&other=123",
	"email+tag@sub.domain.co.uk",
	"C:\\Users\\Admin\\Documents",
	"emoji: 😀🎉🔥💯 and symbols: ©®™",
	"zero\x00width",
	"",  // empty string
	" ", // single space
	"   leading and trailing spaces   ",
}

// Large text generator for articles
func generateLargeText(targetBytes int) string {
	words := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing",
		"technology", "innovation", "development", "software", "engineering",
		"cloud", "computing", "artificial", "intelligence", "machine", "learning",
	}
	var sb strings.Builder
	for sb.Len() < targetBytes {
		word := words[rng.Intn(len(words))]
		if sb.Len() > 0 {
			if rng.Float32() < 0.1 {
				sb.WriteString(". ")
			} else {
				sb.WriteString(" ")
			}
		}
		sb.WriteString(word)
	}
	return sb.String()
}

func generateMarkdown(paragraphs int) string {
	var sb strings.Builder
	for i := 0; i < paragraphs; i++ {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		// Random heading
		if rng.Float32() < 0.3 {
			level := rng.Intn(3) + 1
			sb.WriteString(strings.Repeat("#", level))
			sb.WriteString(" Heading Level ")
			sb.WriteString(strconv.Itoa(level))
			sb.WriteString("\n\n")
		}
		// Paragraph with formatting
		sb.WriteString(generateLargeText(200 + rng.Intn(300)))
		// Random code block
		if rng.Float32() < 0.2 {
			sb.WriteString("\n\n```go\nfunc example() {\n    return nil\n}\n```\n")
		}
		// Random list
		if rng.Float32() < 0.2 {
			sb.WriteString("\n\n- Item one\n- Item two\n- Item three\n")
		}
	}
	return sb.String()
}

// safePrefix returns the first n characters of s, or s if it's shorter than n
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func randomGeopoint() [2]float64 {
	lat := -90.0 + rng.Float64()*180.0
	lon := -180.0 + rng.Float64()*360.0
	return [2]float64{lat, lon}
}

func randomEdgeGeopoint() [2]float64 {
	// Generate points near poles or dateline
	choice := rng.Intn(4)
	switch choice {
	case 0: // Near north pole
		return [2]float64{89.0 + rng.Float64(), -180.0 + rng.Float64()*360.0}
	case 1: // Near south pole
		return [2]float64{-89.0 - rng.Float64(), -180.0 + rng.Float64()*360.0}
	case 2: // Near international date line (positive)
		return [2]float64{-90.0 + rng.Float64()*180.0, 179.0 + rng.Float64()}
	default: // Near international date line (negative)
		return [2]float64{-90.0 + rng.Float64()*180.0, -179.0 - rng.Float64()}
	}
}

func randomStringSlice(options []string, min, max int) []string {
	count := min + rng.Intn(max-min+1)
	result := make([]string, 0, count)
	used := make(map[int]bool)
	for len(result) < count && len(used) < len(options) {
		idx := rng.Intn(len(options))
		if !used[idx] {
			used[idx] = true
			result = append(result, options[idx])
		}
	}
	return result
}

func randomTimestamp(baseYear int, rangeYears int) int64 {
	base := time.Date(baseYear, 1, 1, 0, 0, 0, 0, time.UTC)
	offset := time.Duration(rng.Int63n(int64(rangeYears) * 365 * 24 * int64(time.Hour)))
	return base.Add(offset).Unix()
}

func generateProduct(id int) map[string]interface{} {
	doc := map[string]interface{}{
		"id":            fmt.Sprintf("prod_%06d", id),
		"name":          fmt.Sprintf("Product %d - %s", id, unicodeNames[rng.Intn(len(unicodeNames))]),
		"description":   generateLargeText(100 + rng.Intn(400)),
		"brand":         brands[rng.Intn(len(brands))],
		"categories":    randomStringSlice(categories, 1, 4),
		"tags":          randomStringSlice([]string{"new", "sale", "trending", "bestseller", "limited", "exclusive"}, 0, 4),
		"price":         math.Round(rng.Float64()*1000*100) / 100,
		"reviews_count": rng.Intn(10000),
		"in_stock":      rng.Float32() > 0.2,
		"stock_quantity": rng.Intn(1000),
		"sku":           fmt.Sprintf("SKU-%s-%06d", safePrefix(brands[rng.Intn(len(brands))], 3), id),
		"created_at":    randomTimestamp(2020, 4),
	}

	// Optional fields
	if rng.Float32() > 0.3 {
		doc["original_price"] = doc["price"].(float64) * (1.0 + rng.Float64()*0.5)
	}
	if rng.Float32() > 0.2 {
		doc["rating"] = math.Round(rng.Float64()*5*10) / 10
	}
	if rng.Float32() > 0.4 {
		geo := randomGeopoint()
		doc["location"] = geo
	}
	if rng.Float32() > 0.3 {
		doc["updated_at"] = doc["created_at"].(int64) + rng.Int63n(86400*365)
	}

	return doc
}

func generateUser(id int) map[string]interface{} {
	name := unicodeNames[rng.Intn(len(unicodeNames))]
	locale := locales[rng.Intn(len(locales))]

	doc := map[string]interface{}{
		"id":              fmt.Sprintf("user_%06d", id),
		"username":        fmt.Sprintf("user%d", id),
		"display_name":    name,
		"email":           fmt.Sprintf("user%d@example.com", id),
		"locale":          locale,
		"timezone":        timezones[rng.Intn(len(timezones))],
		"roles":           randomStringSlice([]string{"user", "admin", "moderator", "contributor", "viewer"}, 1, 3),
		"verified":        rng.Float32() > 0.3,
		"follower_count":  rng.Intn(100000),
		"following_count": rng.Intn(5000),
		"joined_at":       randomTimestamp(2015, 9),
	}

	// Optional fields
	if rng.Float32() > 0.4 {
		doc["bio"] = fmt.Sprintf("Bio for %s: %s", name, generateLargeText(50+rng.Intn(200)))
	}
	if rng.Float32() > 0.5 {
		doc["permissions"] = randomStringSlice([]string{"read", "write", "delete", "admin", "moderate"}, 1, 4)
	}
	if rng.Float32() > 0.3 {
		doc["age"] = 18 + rng.Intn(62)
	}
	if rng.Float32() > 0.4 {
		doc["reputation_score"] = math.Round(rng.Float64()*10000*100) / 100
	}
	if rng.Float32() > 0.5 {
		geo := randomGeopoint()
		doc["location"] = geo
	}
	if rng.Float32() > 0.2 {
		doc["last_active_at"] = doc["joined_at"].(int64) + rng.Int63n(86400*365*5)
	}

	return doc
}

func generateArticle(id int) map[string]interface{} {
	category := articleCategories[rng.Intn(len(articleCategories))]
	author := unicodeNames[rng.Intn(len(unicodeNames))]
	title := fmt.Sprintf("Article %d: %s Insights from %s", id, category, author)

	// Vary content size - some articles are very large
	contentSize := 500 + rng.Intn(2000)
	if rng.Float32() < 0.05 {
		// 5% of articles are very large (up to 100KB)
		contentSize = 50000 + rng.Intn(50000)
	}

	wordCount := contentSize / 5
	readingTime := wordCount / 200
	if readingTime < 1 {
		readingTime = 1
	}

	doc := map[string]interface{}{
		"id":                   fmt.Sprintf("article_%06d", id),
		"title":                title,
		"slug":                 fmt.Sprintf("article-%d-%s", id, strings.ToLower(strings.ReplaceAll(category, " ", "-"))),
		"content":              generateLargeText(contentSize),
		"author_id":            fmt.Sprintf("user_%06d", rng.Intn(10000)),
		"author_name":          author,
		"category":             category,
		"tags":                 randomStringSlice([]string{"tutorial", "news", "opinion", "review", "guide", "analysis", "interview"}, 1, 5),
		"reading_time_minutes": int32(readingTime),
		"word_count":           int32(wordCount),
		"view_count":           int64(rng.Intn(1000000)),
		"like_count":           rng.Intn(50000),
		"comment_count":        rng.Intn(1000),
		"is_featured":          rng.Float32() < 0.1,
		"is_published":         rng.Float32() > 0.1,
		"language":             []string{"en", "zh", "ja", "ko", "ar", "ru", "de", "fr", "es", "pt"}[rng.Intn(10)],
		"published_at":         randomTimestamp(2018, 6),
	}

	// Optional fields
	if rng.Float32() > 0.3 {
		doc["excerpt"] = generateLargeText(100 + rng.Intn(200))
	}
	if rng.Float32() > 0.4 {
		doc["subcategory"] = fmt.Sprintf("%s Subcategory", category)
	}
	if rng.Float32() > 0.5 {
		doc["keywords"] = randomStringSlice([]string{"trending", "viral", "breaking", "exclusive", "featured"}, 1, 5)
	}
	if rng.Float32() > 0.3 {
		doc["content_markdown"] = generateMarkdown(3 + rng.Intn(5))
	}
	if rng.Float32() > 0.2 {
		doc["updated_at"] = doc["published_at"].(int64) + rng.Int63n(86400*365*2)
	}

	return doc
}

func generateEvent(id int) map[string]interface{} {
	eventType := eventTypes[rng.Intn(len(eventTypes))]
	startTime := randomTimestamp(2024, 3)
	duration := int64(3600 * (1 + rng.Intn(8))) // 1-8 hours

	doc := map[string]interface{}{
		"id":               fmt.Sprintf("event_%06d", id),
		"name":             fmt.Sprintf("%s: %s Edition %d", eventType, unicodeNames[rng.Intn(len(unicodeNames))], id),
		"description":      generateLargeText(200 + rng.Intn(500)),
		"venue_name":       fmt.Sprintf("Venue %d", rng.Intn(1000)),
		"venue_address":    fmt.Sprintf("%d Main Street, City %d", rng.Intn(9999), rng.Intn(100)),
		"location":         randomGeopoint(),
		"event_type":       eventType,
		"categories":       randomStringSlice(categories, 1, 3),
		"organizer_id":     fmt.Sprintf("user_%06d", rng.Intn(10000)),
		"organizer_name":   unicodeNames[rng.Intn(len(unicodeNames))],
		"registered_count": rng.Intn(10000),
		"is_free":          rng.Float32() < 0.3,
		"is_virtual":       rng.Float32() < 0.4,
		"timezone":         timezones[rng.Intn(len(timezones))],
		"start_timestamp":  startTime,
		"end_timestamp":    startTime + duration,
		"created_at":       startTime - int64(86400*(7+rng.Intn(90))),
	}

	// Optional fields
	if rng.Float32() > 0.3 {
		doc["capacity"] = 50 + rng.Intn(9950)
	}
	if !doc["is_free"].(bool) {
		doc["ticket_price"] = math.Round(rng.Float64()*500*100) / 100
	}
	if rng.Float32() > 0.4 {
		doc["registration_deadline"] = startTime - int64(86400*(1+rng.Intn(30)))
	}

	return doc
}

func generateEdgeCase(id int) map[string]interface{} {
	doc := map[string]interface{}{
		"id":             fmt.Sprintf("edge_%06d", id),
		"test_case_name": fmt.Sprintf("Edge Case %d", id),
		"unicode_text":   unicodeNames[id%len(unicodeNames)],
		"float_precision": 0.1 + 0.2, // Famous floating point issue
		"max_int32":      int32(2147483647 - id),
		"large_int64":    int64(9223372036854775807 - int64(id)),
		"bool_field":     id%2 == 0,
		"category":       fmt.Sprintf("edge_category_%d", id%10),
		"created_at":     randomTimestamp(2020, 4),
	}

	// Cycle through special characters
	if id < len(specialCharsStrings) {
		doc["special_chars_text"] = specialCharsStrings[id]
	} else if rng.Float32() > 0.3 {
		doc["special_chars_text"] = specialCharsStrings[rng.Intn(len(specialCharsStrings))]
	}

	// Empty string allowed
	if rng.Float32() > 0.5 {
		doc["empty_allowed_string"] = ""
	} else if rng.Float32() > 0.3 {
		doc["empty_allowed_string"] = generateLargeText(10 + rng.Intn(50))
	}

	// Large text field (some very large)
	if rng.Float32() > 0.7 {
		size := 1000 + rng.Intn(99000) // Up to ~100KB
		doc["large_text_field"] = generateLargeText(size)
	}

	// Long array
	if rng.Float32() > 0.6 {
		count := 100 + rng.Intn(900) // 100-1000 elements
		arr := make([]string, count)
		for i := 0; i < count; i++ {
			arr[i] = fmt.Sprintf("element_%d", i)
		}
		doc["long_array"] = arr
	}

	// Edge geopoints
	if rng.Float32() > 0.5 {
		doc["near_pole_location"] = randomEdgeGeopoint()
	}
	if rng.Float32() > 0.5 {
		doc["near_dateline_location"] = randomEdgeGeopoint()
	}

	// Optional fields (randomly present or absent)
	if rng.Float32() > 0.5 {
		doc["optional_bool"] = rng.Float32() > 0.5
	}
	if rng.Float32() > 0.5 {
		doc["optional_int"] = rng.Intn(1000000)
	}
	if rng.Float32() > 0.5 {
		doc["optional_float"] = math.Round(rng.Float64()*1000000*100) / 100
	}
	if rng.Float32() > 0.5 {
		doc["optional_array"] = randomStringSlice([]string{"opt1", "opt2", "opt3", "opt4", "opt5"}, 0, 5)
	}

	return doc
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <collection> <count>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Collections: products, users, articles, events, edge_cases\n")
		os.Exit(1)
	}

	collection := os.Args[1]
	count, err := strconv.Atoi(os.Args[2])
	if err != nil || count <= 0 {
		fmt.Fprintf(os.Stderr, "Error: count must be a positive integer\n")
		os.Exit(1)
	}

	// Reset RNG with seed for reproducibility per collection
	seed := int64(42)
	switch collection {
	case "products":
		seed = 1001
	case "users":
		seed = 2002
	case "articles":
		seed = 3003
	case "events":
		seed = 4004
	case "edge_cases":
		seed = 5005
	}
	rng = rand.New(rand.NewSource(seed))

	encoder := json.NewEncoder(os.Stdout)

	for i := 0; i < count; i++ {
		var doc map[string]interface{}

		switch collection {
		case "products":
			doc = generateProduct(i)
		case "users":
			doc = generateUser(i)
		case "articles":
			doc = generateArticle(i)
		case "events":
			doc = generateEvent(i)
		case "edge_cases":
			doc = generateEdgeCase(i)
		default:
			fmt.Fprintf(os.Stderr, "Unknown collection: %s\n", collection)
			os.Exit(1)
		}

		if err := encoder.Encode(doc); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding document %d: %v\n", i, err)
			os.Exit(1)
		}
	}
}
