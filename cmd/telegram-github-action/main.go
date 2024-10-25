package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
}

// Version returns a formatted string with application version details.
func (ActionInputs) Version() string {
	return fmt.Sprintf("Version: %s %s\nBuildTime: %s\n%s\n", Revision, Version, StartTime.Format("2006-01-02"), GoVersion)
}

func main() {
	var args ActionInputs
	arg.MustParse(&args)

	log.Println("Starting Telegram GitHub Action")

	// Validate ParseMode
	if args.ParseMode != "" && args.ParseMode != "markdown" && args.ParseMode != "html" {
		log.Fatalf("Invalid ParseMode: %v. Allowed values are 'html' or 'markdown'.\n", args.ParseMode)
	}

	// Decide whether to send a custom message or call getMe API
	if args.Message == "" {
		log.Println("No custom message provided, calling the getMe API")
		if err := callTelegramAPI(args.Token, "getMe", nil); err != nil {
			log.Fatalf("Error calling getMe API: %v\n", err)
		}
	} else {
		log.Printf("Sending custom message to chat ID %s\n", args.To)
		// Convert args.To to int64 to handle negative chat IDs
		toInt, err := strconv.ParseInt(args.To, 10, 64)
		if err != nil {
			log.Fatalf("Invalid chat ID: %v\n", err)
		}

		messagePayload := TelegramMessage{
			ChatID:                toInt,
			Text:                  args.Message,
			ParseMode:             args.ParseMode,
			DisableWebPagePreview: args.DisableWebPagePreview,
			DisableNotification:   args.DisableNotification,
			ProtectContent:        args.ProtectContent,
		}

		if err := callTelegramAPI(args.Token, "sendMessage", messagePayload); err != nil {
			log.Fatalf("Error sending Telegram message: %v\n", err)
		}
		log.Println("Message sent successfully")
	}
}

// callTelegramAPI handles calling the Telegram Bot API with the specified method and payload.
func callTelegramAPI(token, method string, payload interface{}) error {
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
			log.Printf("Warning: error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response from Telegram API: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	log.Printf("%s API response: %s\n", method, string(responseBody))
	return nil
}

