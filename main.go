package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mozillazg/go-unidecode"
)

// --- CONFIGURATION ---
var (
	serverURL string
	token     string
	jsonFile  string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to upload emojis to Mattermost from a JSON file.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -s, --server string\n")
		fmt.Fprintf(os.Stderr, "        Mattermost server URL without trailing slash (required)\n")
		fmt.Fprintf(os.Stderr, "  -t, --token string\n")
		fmt.Fprintf(os.Stderr, "        Personal Access Token (required)\n")
		fmt.Fprintf(os.Stderr, "  -f, --file string\n")
		fmt.Fprintf(os.Stderr, "        Path to your source JSON file (required)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -server https://mattermost.example.com -token TOKEN -file emoji.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -s https://mattermost.example.com -t TOKEN -f emoji.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFor more information, see: https://github.com/formatCvt/mattermost-emoji-uploader\n")
	}

	flag.StringVar(&serverURL, "server", "", "Mattermost server URL without trailing slash (required)")
	flag.StringVar(&serverURL, "s", "", "Mattermost server URL without trailing slash (required)")
	flag.StringVar(&token, "token", "", "Personal Access Token (required)")
	flag.StringVar(&token, "t", "", "Personal Access Token (required)")
	flag.StringVar(&jsonFile, "file", "", "Path to your source JSON file (required)")
	flag.StringVar(&jsonFile, "f", "", "Path to your source JSON file (required)")
}

type EmojiMap map[string]string

type UserInfo struct {
	ID string `json:"id"`
}

func main() {
	flag.Parse()

	// Validate required flags
	if serverURL == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: -server/-s flag is required\n")
		flag.Usage()
		os.Exit(1)
	}
	if token == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: -token/-t flag is required\n")
		flag.Usage()
		os.Exit(1)
	}
	if jsonFile == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: -file/-f flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// 1. Read the JSON source file
	file, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("❌ Error reading file: %v\n", err)
		return
	}

	var emojis EmojiMap
	if err := json.Unmarshal(file, &emojis); err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get user ID from token
	userID, err := getUserID(client, serverURL, token)
	if err != nil {
		fmt.Printf("❌ Error getting user ID: %v\n", err)
		return
	}

	fmt.Printf("🚀 Starting import of %d emojis...\n\n", len(emojis))

	for originalName, url := range emojis {
		// Clean the name to meet Mattermost requirements (latin, lowercase, no special chars)
		safeName := sanitizeEmojiName(originalName)

		fmt.Printf("Processing: [:%s:] -> [:%s:]... ", originalName, safeName)

		// Skip aliases (they reference existing emojis, not image URLs)
		if strings.HasPrefix(url, "alias:") {
			fmt.Println("⏭️  Skipped (alias - references existing emoji)")
			continue
		}

		// 2. Download the image into a temporary memory buffer
		imgData, contentType, err := downloadImage(client, url)
		if err != nil {
			fmt.Printf("❌ Download error: %v\n", err)
			continue
		}

		// 3. Upload the buffer to Mattermost
		err = uploadToMattermost(client, serverURL, token, safeName, imgData, contentType, userID)
		if err != nil {
			// Check if emoji already exists (Mattermost returns 400 for duplicates)
			if strings.Contains(err.Error(), "400") {
				fmt.Println("⚠️  Skipped (already exists or invalid name)")
			} else {
				fmt.Printf("❌ Upload error: %v\n", err)
			}
		} else {
			fmt.Println("✅ Success!")
		}

		// Brief pause to avoid triggering rate limits
		time.Sleep(200 * time.Millisecond)
	}
}

// sanitizeEmojiName converts names to Mattermost-compatible format
func sanitizeEmojiName(name string) string {
	// Transliterate non-latin characters (e.g., "жду" -> "zhdu")
	name = unidecode.Unidecode(name)
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace spaces with dashes
	name = strings.ReplaceAll(name, " ", "-")
	// Remove all forbidden characters (anything not a-z, 0-9, - or _)
	reg := regexp.MustCompile(`[^a-z0-9\-_]+`)
	name = reg.ReplaceAllString(name, "")
	// Truncate to Mattermost limit (64 chars)
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

// downloadImage fetches the image from Slack/external URL
func downloadImage(client *http.Client, url string) ([]byte, string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// getUserID retrieves the user ID from the token
func getUserID(client *http.Client, serverURL, token string) (string, error) {
	req, err := http.NewRequest("GET", serverURL+"/api/v4/users/me", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", err
	}

	return userInfo.ID, nil
}

// uploadToMattermost performs the multipart/form-data POST request
func uploadToMattermost(client *http.Client, serverURL, token, name string, imgData []byte, contentType string, creatorID string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 'emoji' field containing JSON metadata with creator_id
	emojiMeta := fmt.Sprintf(`{"name":"%s","creator_id":"%s"}`, name, creatorID)
	_ = writer.WriteField("emoji", emojiMeta)

	// 'image' field containing binary data
	// Detect extension based on Content-Type for the filename parameter
	ext := ".png"
	if contentType == "image/gif" {
		ext = ".gif"
	} else if contentType == "image/jpeg" {
		ext = ".jpg"
	}

	part, err := writer.CreateFormFile("image", name+ext)
	if err != nil {
		return err
	}
	_, err = part.Write(imgData)
	if err != nil {
		return err
	}

	writer.Close()

	req, err := http.NewRequest("POST", serverURL+"/api/v4/emoji", body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Mattermost may return either 200 (OK) or 201 (Created) for successful emoji creation
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
