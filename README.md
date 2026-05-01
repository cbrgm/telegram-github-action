# Telegram GitHub Action

**Send messages and media to a Telegram chat from GitHub Actions.**

[![GitHub release](https://img.shields.io/github/release/cbrgm/telegram-github-action.svg)](https://github.com/cbrgm/telegram-github-action)
[![Go Report Card](https://goreportcard.com/badge/github.com/cbrgm/telegram-github-action)](https://goreportcard.com/report/github.com/cbrgm/telegram-github-action)
[![go-lint-test](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-lint-test.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-lint-test.yml)
[![go-binaries](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-binaries.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-binaries.yml)
[![container](https://github.com/cbrgm/telegram-github-action/actions/workflows/container.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/container.yml)

## Inputs

- `token`: **Required** - Telegram bot’s authorization token. Use GitHub secrets.
- `to`: **Required** - Unique identifier or username of the target Telegram chat.
- `thread-id`: (Optional) Thread identifier in a Telegram supergroup.
    - The message won’t be sent if `to` isn’t a supergroup and `thread-id` is set.
- `message`: Optional - Text message to send. Used as caption when `media` is provided.
- `parse-mode`: Optional - Mode for parsing text entities (`markdown` or `html`).
- `media`: Optional - Media to send. Can be a local file path, an HTTP URL, or a Telegram `file_id`.
- `media-type`: Optional - Type of media. One of: `photo`, `video`, `audio`, `document`, `animation`, `voice`, `sticker`. Required when `media` is set.
- `disable-web-page-preview`: Optional - Disables link previews.
- `disable-notification`: Optional - Sends message silently.
- `protect-content`: Optional - Protects message content from forwarding/saving.
- `dry-run`: Optional - If set, do not send a real message but print the details instead.

### Workflow Usage

```yaml
name: Send Telegram Message

on: [push]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Send Telegram Message
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          message: "New commit pushed to repository"
```

#### Example Workflow: Inline Messages and Variable Templating

This workflow triggers a Telegram message notification when a new tag is published in the repository.

```yaml
name: Inline Message Workflow

on:
  push:
    branches: [main]

jobs:
  send-inline-message:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Send Inline Telegram Message
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          message: |
            New commit by ${{ github.actor }}!
            Commit: ${{ github.event.head_commit.message }}
            Repository: ${{ github.repository }}
            View changes: https://github.com/${{ github.repository }}/commit/${{ github.sha }}
```

### Example Workflow: Notification on New GitHub Release

This workflow triggers a Telegram message notification when a new release is published in the repository.

```yaml
name: Release Notification

on:
  release:
    types: [published]

jobs:
  notify-on-release:
    runs-on: ubuntu-latest
    steps:
      - name: Send Telegram Notification on New Release
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          message: |
            🚀 New Release Published!
            Release Name: ${{ github.event.release.name }}
            Tag: ${{ github.event.release.tag_name }}
            Actor: ${{ github.actor }}
            Repository: ${{ github.repository }}
            Check it out: ${{ github.event.release.html_url }}j

```

### Example Workflow: Send a Photo via URL

```yaml
name: Send Photo

on: [push]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Send Photo to Telegram
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          media: "https://example.com/build-status.png"
          media-type: "photo"
          message: "Latest build status for ${{ github.repository }}"
```

### Example Workflow: Upload a Local File

This workflow uploads a build artifact (e.g. a PDF report or binary) directly from the runner filesystem.

```yaml
name: Upload Build Report

on:
  workflow_run:
    workflows: ["Build"]
    types: [completed]

jobs:
  send-report:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Generate report
        run: echo "Build completed at $(date)" > report.txt

      - name: Send Report to Telegram
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          media: "report.txt"
          media-type: "document"
          message: "Build report attached"
```

### Example Workflow: Send a Video

```yaml
name: Send Demo Video

on:
  release:
    types: [published]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Send Video to Telegram
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          media: "https://example.com/demo.mp4"
          media-type: "video"
          message: "Demo video for release ${{ github.event.release.tag_name }}"
```

### Example Workflow: Send an Animation (GIF)

```yaml
name: CI Status GIF

on:
  pull_request:
    types: [opened]

jobs:
  welcome:
    runs-on: ubuntu-latest
    steps:
      - name: Send Welcome GIF
        uses: cbrgm/telegram-github-action@v1
        with:
          token: ${{ secrets.TELEGRAM_TOKEN }}
          to: ${{ secrets.TELEGRAM_CHAT_ID }}
          media: "https://example.com/welcome.gif"
          media-type: "animation"
          message: "New PR opened by ${{ github.actor }}"
```

### Supported Media Types

| Type | API Method | Description | Max Size |
|------|-----------|-------------|----------|
| `photo` | sendPhoto | Images (JPEG, PNG, etc.) | 10 MB |
| `video` | sendVideo | Video files (MPEG4 recommended) | 50 MB |
| `audio` | sendAudio | Audio for music players (MP3, M4A) | 50 MB |
| `document` | sendDocument | Any file type | 50 MB |
| `animation` | sendAnimation | GIFs or H.264 videos without sound | 50 MB |
| `voice` | sendVoice | Voice messages (OGG/OPUS, MP3, M4A) | 50 MB |
| `sticker` | sendSticker | Stickers (WEBP, TGS, WEBM). No caption support. | 50 MB |

The `media` input accepts three formats:
- **Local file path**: Uploads the file from the runner (e.g. `./artifacts/screenshot.png`)
- **HTTP URL**: Telegram downloads the file from the URL (e.g. `https://example.com/image.jpg`)
- **Telegram file_id**: Resends a file already on Telegram's servers (e.g. `AgACAgIAAxk...`)

### Creating a Telegram Bot and Obtaining a Token

1. Chat with [BotFather](https://t.me/botfather) on Telegram.
2. Follow prompts to name your bot and get a token.
3. Store the token as a GitHub secret (`TELEGRAM_TOKEN`).

### Obtaining the Chat ID
Run the following command:
```bash
curl https://api.telegram.org/bot<token>/getUpdates
```
Replace <token> with your bot's token to find your `chat_id` (`TELEGRAM_CHAT_ID`).


### Obtaining the Thread ID
The easiest way to get the thread ID is as follows: Post a message in that thread, then right-click it and choose Copy Message Link. Paste it onto a scratchpad and you will notice that it has the following structure https://t.me/c/XXXXXXXXXX/YY/ZZ . The thread ID is YY (integer).
### Local Development

You can build this action from source using `Go`:

```bash
make build
```

## Contributing & License

* **Contributions Welcome!**: Interested in improving or adding features? Check our [Contributing Guide](https://github.com/cbrgm/telegram-github-action/blob/main/CONTRIBUTING.md) for instructions on submitting changes and setting up development environment.
* **Open-Source & Free**: Developed in my spare time, available for free under [Apache 2.0 License](https://github.com/cbrgm/telegram-github-action/blob/main/LICENSE). License details your rights and obligations.
* **Your Involvement Matters**: Code contributions, suggestions, feedback crucial for improvement and success. Let's maintain it as a useful resource for all 🌍.
