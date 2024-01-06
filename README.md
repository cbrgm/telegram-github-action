# Telegram GitHub Action

**Send (text) messages to a Telegram chat from GitHub Actions.**

[![GitHub release](https://img.shields.io/github/release/cbrgm/telegram-github-action.svg)](https://github.com/cbrgm/telegram-github-action)
[![Go Report Card](https://goreportcard.com/badge/github.com/cbrgm/telegram-github-action)](https://goreportcard.com/report/github.com/cbrgm/telegram-github-action)
[![go-lint-test](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-lint-test.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-lint-test.yml)
[![go-binaries](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-binaries.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/go-binaries.yml)
[![container](https://github.com/cbrgm/telegram-github-action/actions/workflows/container.yml/badge.svg)](https://github.com/cbrgm/telegram-github-action/actions/workflows/container.yml)

## Inputs

- `token`: **Required** - Telegram bot's authorization token. Use GitHub secrets.
- `to`: **Required** - Unique identifier or username of the target Telegram chat.
- `message`: Optional - Text message to send. If omitted, bot's information is fetched.
- `parse-mode`: Optional - Mode for parsing text entities (`markdown` or `html`).
- `disable-web-page-preview`: Optional - Disables link previews.
- `disable-notification`: Optional - Sends message silently.
- `protect-content`: Optional - Protects message content from forwarding/saving.

### Workflow Usage

```yaml
name: Send Telegram Message

on: [push]

jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - name: Send Telegram Message
        uses: cbrgm/telegram-github-action@v1.0
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
        uses: cbrgm/telegram-github-action@v1.0
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
        uses: cbrgm/telegram-github-action@v1.0
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

### Local Development

You can build this action from source using `Go`:

```bash
make build
```

## Contributing & License

We welcome and value your contributions to this project! 👍 If you're interested in making improvements or adding features, please refer to our [Contributing Guide](https://github.com/cbrgm/telegram-github-action/blob/main/CONTRIBUTING.md). This guide provides comprehensive instructions on how to submit changes, set up your development environment, and more.

Please note that this project is developed in my spare time and is available for free 🕒💻. As an open-source initiative, it is governed by the [Apache 2.0 License](https://github.com/cbrgm/telegram-github-action/blob/main/LICENSE). This license outlines your rights and obligations when using, modifying, and distributing this software.

Your involvement, whether it's through code contributions, suggestions, or feedback, is crucial for the ongoing improvement and success of this project. Together, we can ensure it remains a useful and well-maintained resource for everyone 🌍.
