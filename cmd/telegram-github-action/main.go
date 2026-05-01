package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

var (
	Version   string
	Revision  string
	GoVersion = runtime.Version()
	StartTime = time.Now()
)

type TelegramMessage struct {
	ChatID                int64  `json:"chat_id"`
	MessageThreadID       *int64 `json:"message_thread_id,omitempty"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
	ProtectContent        bool   `json:"protect_content,omitempty"`
}

type ActionInputs struct {
	Token                 string `arg:"--token,env:TELEGRAM_TOKEN,required"`
	To                    string `arg:"--to,env:TELEGRAM_TO,required"`
	MessageThreadID       string `arg:"--thread-id,env:TELEGRAM_THREAD_ID"`
	Message               string `arg:"--message,env:MESSAGE"`
	ParseMode             string `arg:"--parse-mode,env:PARSE_MODE"`
	Media                 string `arg:"--media,env:TELEGRAM_MEDIA"`
	MediaType             string `arg:"--media-type,env:TELEGRAM_MEDIA_TYPE"`
	DisableWebPagePreview bool   `arg:"--disable-web-page-preview,env:DISABLE_WEB_PAGE_PREVIEW"`
	DisableNotification   bool   `arg:"--disable-notification,env:DISABLE_NOTIFICATION"`
	ProtectContent        bool   `arg:"--protect-content,env:PROTECT_CONTENT"`
	DryRun                bool   `arg:"--dry-run" help:"If set, do not send a real message but print the details instead"`
}

func (ActionInputs) Version() string {
	return fmt.Sprintf("Version: %s %s\nBuildTime: %s\n%s\n", Revision, Version, StartTime.Format("2006-01-02"), GoVersion)
}

var validMediaTypes = map[string]string{
	"photo":     "sendPhoto",
	"video":     "sendVideo",
	"audio":     "sendAudio",
	"document":  "sendDocument",
	"animation": "sendAnimation",
	"voice":     "sendVoice",
	"sticker":   "sendSticker",
}

var mediaFieldName = map[string]string{
	"photo":     "photo",
	"video":     "video",
	"audio":     "audio",
	"document":  "document",
	"animation": "animation",
	"voice":     "voice",
	"sticker":   "sticker",
}

func main() {
	var args ActionInputs
	arg.MustParse(&args)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("Starting Telegram GitHub Action")

	if args.ParseMode != "" && args.ParseMode != "markdown" && args.ParseMode != "html" {
		logger.Error("Invalid ParseMode", slog.String("ParseMode", args.ParseMode))
		os.Exit(1)
	}

	toInt, err := strconv.ParseInt(args.To, 10, 64)
	if err != nil {
		logger.Error("Invalid chat ID", slog.String("ChatID", args.To), slog.Any("error", err))
		os.Exit(1)
	}

	var threadID *int64
	if trimmedThreadID := strings.TrimSpace(args.MessageThreadID); trimmedThreadID != "" {
		parsedThreadID, err := strconv.ParseInt(trimmedThreadID, 10, 64)
		if err != nil {
			logger.Error("Could not parse MessageThreadID into an integer value", slog.String("MessageThreadID", args.MessageThreadID), slog.Any("error", err))
			os.Exit(1)
		}
		threadID = &parsedThreadID
	}

	media := strings.TrimSpace(args.Media)
	mediaType := strings.TrimSpace(args.MediaType)

	if media != "" && mediaType == "" {
		logger.Error("media-type is required when media is provided")
		os.Exit(1)
	}

	if mediaType != "" && media == "" {
		logger.Error("media is required when media-type is provided")
		os.Exit(1)
	}

	if mediaType != "" {
		if _, ok := validMediaTypes[mediaType]; !ok {
			logger.Error("Invalid media-type", slog.String("media-type", mediaType))
			os.Exit(1)
		}
	}

	if media == "" && args.Message == "" {
		logger.Error("Either message or media must be provided")
		os.Exit(1)
	}

	if args.DryRun {
		logger.Info("Dry run enabled, message will not be sent",
			slog.String("Message", args.Message),
			slog.String("ParseMode", args.ParseMode),
			slog.String("Media", media),
			slog.String("MediaType", mediaType),
			slog.Bool("DisableWebPagePreview", args.DisableWebPagePreview),
			slog.Bool("DisableNotification", args.DisableNotification),
			slog.Bool("ProtectContent", args.ProtectContent),
		)
		return
	}

	if media != "" {
		logger.Info("Sending media message", slog.String("media-type", mediaType))
		err := sendMedia(logger, args.Token, toInt, threadID, media, mediaType, args.Message, args.ParseMode, args.DisableNotification, args.ProtectContent)
		if err != nil {
			logger.Error("Error sending media", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Media sent successfully")
	} else {
		messagePayload := TelegramMessage{
			ChatID:                toInt,
			MessageThreadID:       threadID,
			Text:                  args.Message,
			ParseMode:             args.ParseMode,
			DisableWebPagePreview: args.DisableWebPagePreview,
			DisableNotification:   args.DisableNotification,
			ProtectContent:        args.ProtectContent,
		}

		logger.Info("Sending text message")
		if err := callTelegramAPI(logger, args.Token, "sendMessage", messagePayload); err != nil {
			logger.Error("Error sending Telegram message", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Message sent successfully")
	}
}

func sendMedia(logger *slog.Logger, token string, chatID int64, threadID *int64, media, mediaType, caption, parseMode string, disableNotification, protectContent bool) error {
	method := validMediaTypes[mediaType]
	fieldName := mediaFieldName[mediaType]
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)

	isFilePath := isLocalFile(media)

	if isFilePath {
		return sendMediaMultipart(logger, apiURL, chatID, threadID, media, fieldName, mediaType, caption, parseMode, disableNotification, protectContent)
	}

	return sendMediaJSON(logger, apiURL, chatID, threadID, media, fieldName, mediaType, caption, parseMode, disableNotification, protectContent)
}

func isLocalFile(media string) bool {
	if strings.HasPrefix(media, "http://") || strings.HasPrefix(media, "https://") {
		return false
	}
	_, err := os.Stat(media)
	return err == nil
}

func sendMediaJSON(logger *slog.Logger, apiURL string, chatID int64, threadID *int64, media, fieldName, mediaType, caption, parseMode string, disableNotification, protectContent bool) error {
	payload := map[string]any{
		"chat_id": chatID,
		fieldName: media,
	}

	if threadID != nil {
		payload["message_thread_id"] = *threadID
	}

	if caption != "" && mediaType != "sticker" {
		payload["caption"] = caption
	}
	if parseMode != "" && mediaType != "sticker" {
		payload["parse_mode"] = parseMode
	}
	if disableNotification {
		payload["disable_notification"] = true
	}
	if protectContent {
		payload["protect_content"] = true
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return doRequest(logger, req)
}

func sendMediaMultipart(logger *slog.Logger, apiURL string, chatID int64, threadID *int64, filePath, fieldName, mediaType, caption, parseMode string, disableNotification, protectContent bool) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("Error closing file", slog.Any("error", err))
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("error creating form file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return fmt.Errorf("error copying file data: %w", err)
	}

	if err := writer.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return fmt.Errorf("error writing chat_id field: %w", err)
	}

	if threadID != nil {
		if err := writer.WriteField("message_thread_id", strconv.FormatInt(*threadID, 10)); err != nil {
			return fmt.Errorf("error writing message_thread_id field: %w", err)
		}
	}

	if caption != "" && mediaType != "sticker" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("error writing caption field: %w", err)
		}
	}
	if parseMode != "" && mediaType != "sticker" {
		if err := writer.WriteField("parse_mode", parseMode); err != nil {
			return fmt.Errorf("error writing parse_mode field: %w", err)
		}
	}
	if disableNotification {
		if err := writer.WriteField("disable_notification", "true"); err != nil {
			return fmt.Errorf("error writing disable_notification field: %w", err)
		}
	}
	if protectContent {
		if err := writer.WriteField("protect_content", "true"); err != nil {
			return fmt.Errorf("error writing protect_content field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return doRequest(logger, req)
}

func doRequest(logger *slog.Logger, req *http.Request) error {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Telegram API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("Error closing response body", slog.Any("error", err))
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response from Telegram API: %s, body: %s", resp.Status, string(responseBody))
	}

	logger.Info("API response", slog.String("response", string(responseBody)))
	return nil
}

func callTelegramAPI(logger *slog.Logger, token, method string, payload any) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("error marshaling payload: %w", err)
		}
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return doRequest(logger, req)
}
