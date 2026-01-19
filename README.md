# Mattermost Emoji Uploader

A command-line tool written in Go that automatically uploads emojis to Mattermost from a JSON file containing emoji names and their image URLs. Perfect for migrating emojis from Slack or other platforms to Mattermost.

## Features

- 🚀 **Batch Upload**: Upload multiple emojis in one run
- 🔄 **Automatic Name Sanitization**: Converts emoji names to Mattermost-compatible format
  - Transliterates non-Latin characters (e.g., "жду" → "zhdu")
  - Converts to lowercase
  - Replaces spaces with dashes
  - Removes special characters
  - Truncates to 64 characters (Mattermost limit)
- 🌐 **URL Support**: Downloads images from any accessible URL
- ⚡ **Rate Limiting**: Built-in delays to avoid API rate limits
- ✅ **Error Handling**: Gracefully handles duplicates and errors

## Requirements

- Go 1.22.2 or higher
- Mattermost server with API access
- Personal Access Token

## Installation

```bash
git clone <repository-url>
cd mattermost-emoji-uploader
go mod download
go build -o mattermost-emoji-uploader
```

## Usage

```bash
./mattermost-emoji-uploader --server <SERVER_URL> --token <TOKEN> --file <JSON_FILE>
```

Or using short flags:

```bash
./mattermost-emoji-uploader -s <SERVER_URL> -t <TOKEN> -f <JSON_FILE>
```

### Required Flags

- `--server` / `-s`: Mattermost server URL without trailing slash (e.g., `https://mattermost.example.com`)
- `--token` / `-t`: Personal Access Token with emoji upload permissions
- `--file` / `-f`: Path to JSON file containing emoji mappings

### Example

Using long flags:
```bash
./mattermost-emoji-uploader \
  --server https://mattermost.example.com \
  --token abc123xyz789 \
  --file emoji.json
```

Using short flags:
```bash
./mattermost-emoji-uploader \
  -s https://mattermost.example.com \
  -t abc123xyz789 \
  -f emoji.json
```

## Exporting Emojis from Slack

To migrate emojis from Slack to Mattermost, you can use [slackdump](https://github.com/rusq/slackdump) - a powerful tool that allows you to export Slack workspace data, including emojis, without admin privileges.

### Step 1: Install slackdump

**On macOS:**
```bash
brew install slackdump
```

**On other systems:**
1. Download the latest release from the [releases page](https://github.com/rusq/slackdump/releases)
2. Unpack the archive
3. Run the executable

### Step 2: Export Emojis from Slack

Run slackdump in emoji download mode:

```bash
slackdump emoji -o emoji.json
```

This will:
- Authenticate with your Slack workspace (using EZ-Login 3000 or browser tokens)
- Download all custom emojis from your Slack workspace
- Save them to `emoji.json` in the format compatible with this tool

For more information about slackdump authentication and options, see the [slackdump documentation](https://github.com/rusq/slackdump).

### Step 3: Upload to Mattermost

Once you have the `emoji.json` file, use this tool to upload emojis to Mattermost:

```bash
./mattermost-emoji-uploader \
  --server https://mattermost.example.com \
  --token your-mattermost-token \
  --file emoji.json
```

Or using short flags:

```bash
./mattermost-emoji-uploader \
  -s https://mattermost.example.com \
  -t your-mattermost-token \
  -f emoji.json
```

## JSON File Format

The JSON file should contain a map of emoji names to image URLs:

```json
{
  "smile": "https://example.com/smile.png",
  "heart": "https://example.com/heart.gif",
  "custom_emoji": "https://example.com/custom.jpg",
  "жду": "https://example.com/zhdu.png",
  "shipit": "alias:squirrel"
}
```

**Note about aliases**: If an emoji value starts with `alias:`, it will be skipped. Aliases are references to existing emojis (common in Slack exports) and don't require image uploads. The tool will display `⏭️ Skipped (alias - references existing emoji)` for such entries.

The tool will automatically sanitize emoji names to meet Mattermost requirements. For example:
- `"жду"` will be converted to `"zhdu"`
- `"My Emoji"` will be converted to `"my-emoji"`
- `"emoji@123"` will be converted to `"emoji123"`

## Getting a Personal Access Token

1. Log in to your Mattermost instance
2. Go to **Account Settings** → **Security** → **Personal Access Tokens**
3. Click **Create Token**
4. Copy the token (you won't be able to see it again)

For more details, see the official documentation on [how to generate a personal access token](https://developers.mattermost.com/integrate/reference/personal-access-token/).

## Supported Image Formats

- PNG (`.png`)
- GIF (`.gif`)
- JPEG (`.jpg`)

The tool automatically detects the image format from the `Content-Type` header.

## Behavior

- **Duplicate Emojis**: If an emoji with the same name already exists, it will be skipped with a warning
- **Invalid Names**: Emojis with invalid names after sanitization will be skipped
- **Download Errors**: Failed downloads are logged and the tool continues with the next emoji
- **Rate Limiting**: A 200ms delay is added between uploads to avoid triggering rate limits

## Output

The tool provides real-time feedback:

```
🚀 Starting import of 10 emojis...

Processing: [:smile:] -> [:smile:]... ✅ Success!
Processing: [:heart:] -> [:heart:]... ✅ Success!
Processing: [:жду:] -> [:zhdu:]... ✅ Success!
Processing: [:duplicate:] -> [:duplicate:]... ⚠️  Skipped (already exists or invalid name)
```

## Error Handling

- Missing required flags: Shows error message and usage information
- Invalid JSON file: Shows parsing error
- Network errors: Logs error and continues with next emoji
- API errors: Shows HTTP status code and error message

## License

[MIT](LICENSE)