package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
)

// Global variables for application metadata.
var (
	Version   string              // Version of the application.
	Revision  string              // Revision or Commit this binary was built from.
	GoVersion = runtime.Version() // GoVersion running this binary.
	StartTime = time.Now()        // StartTime of the application.
)

// TelegramMessage represents the payload for sending messages via the Telegram API.
type TelegramMessage struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
	ProtectContent        bool   `json:"protect_content,omitempty"`
}

// ActionInputs holds the required environment variables for the action.
type ActionInputs struct {
	Token                 string `arg:"--token,env:TELEGRAM_TOKEN,required"`
	To                    string `arg:"--to,env:TELEGRAM_TO,required"`
	Message               string `arg:"--message,env:MESSAGE"`
	ParseMode             string `arg:"--parse-mode,env:PARSE_MODE"`
	DisableWebPagePreview bool   `arg:"--disable-web-page-preview,env:DISABLE_WEB_PAGE_PREVIEW"`
	DisableNotification   bool   `arg:"--disable-notification,env:DISABLE_NOTIFICATION"`
	ProtectContent        bool   `arg:"--protect-content,env:PROTECT_CONTENT"`
	DryRun                bool   `arg:"--dry-run" help:"If set, do not send a real message but print the details instead"`
}

// Version returns a formatted string with application version details.
func (ActionInputs) Version() string {
	return fmt.Sprintf("Version: %s %s\nBuildTime: %s\n%s\n", Revision, Version, StartTime.Format("2006-01-02"), GoVersion)
}

func main() {
	var args ActionInputs
	arg.MustParse(&args)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Telegram GitHub Action")

	// Validate ParseMode
	if args.ParseMode != "" && args.ParseMode != "markdown" && args.ParseMode != "html" {
		logger.Error("Invalid ParseMode", slog.String("ParseMode", args.ParseMode))
		os.Exit(1)
	}

	// Convert args.To to int64 to handle negative chat IDs
	toInt, err := strconv.ParseInt(args.To, 10, 64)
	if err != nil {
		logger.Error("Invalid chat ID", slog.String("ChatID", args.To), slog.Any("error", err))
		os.Exit(1)
	}

	messagePayload := TelegramMessage{
		ChatID:                toInt,
		Text:                  args.Message,
		ParseMode:             args.ParseMode,
		DisableWebPagePreview: args.DisableWebPagePreview,
		DisableNotification:   args.DisableNotification,
		ProtectContent:        args.ProtectContent,
	}

	// Dry-run check
	if args.DryRun {
		logger.Info("Dry run enabled, message will not be sent",
			slog.String("Message", args.Message),
			slog.String("ParseMode", args.ParseMode),
			slog.Bool("DisableWebPagePreview", args.DisableWebPagePreview),
			slog.Bool("DisableNotification", args.DisableNotification),
			slog.Bool("ProtectContent", args.ProtectContent))
		return
	}

	// Actual API call if not in dry run
	logger.Info("Sending custom message")
	if err := callTelegramAPI(logger, args.Token, "sendMessage", messagePayload); err != nil {
		logger.Error("Error sending Telegram message", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("Message sent successfully")
}

// callTelegramAPI handles calling the Telegram Bot API with the specified method and payload.
func callTelegramAPI(logger *slog.Logger, token, method string, payload interface{}) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}
	} else {
		body = nil
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Telegram API: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logger.Warn("Error closing response body", slog.Any("error", err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response from Telegram API: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	logger.Info("API response", slog.String("method", method), slog.String("response", string(responseBody)))
	return nil
}
