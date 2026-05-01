package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsLocalFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-media-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"HTTP URL", "http://example.com/image.jpg", false},
		{"HTTPS URL", "https://example.com/image.jpg", false},
		{"File ID", "AgACAgIAAxkBAAI", false},
		{"Local file", tmpFile.Name(), true},
		{"Non-existent file", "/nonexistent/path/file.jpg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalFile(tt.input)
			if result != tt.expected {
				t.Errorf("isLocalFile(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSendMediaJSON_Photo(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":123}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, 12345, nil, "https://example.com/photo.jpg", "photo", "photo", "Test caption", "markdown", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["chat_id"].(float64) != 12345 {
		t.Errorf("expected chat_id 12345, got %v", receivedBody["chat_id"])
	}
	if receivedBody["photo"] != "https://example.com/photo.jpg" {
		t.Errorf("expected photo URL, got %v", receivedBody["photo"])
	}
	if receivedBody["caption"] != "Test caption" {
		t.Errorf("expected caption 'Test caption', got %v", receivedBody["caption"])
	}
	if receivedBody["parse_mode"] != "markdown" {
		t.Errorf("expected parse_mode 'markdown', got %v", receivedBody["parse_mode"])
	}
}

func TestSendMediaJSON_Video(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":124}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, -100123456, nil, "https://example.com/video.mp4", "video", "video", "Video caption", "html", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["chat_id"].(float64) != -100123456 {
		t.Errorf("expected chat_id -100123456, got %v", receivedBody["chat_id"])
	}
	if receivedBody["video"] != "https://example.com/video.mp4" {
		t.Errorf("expected video URL, got %v", receivedBody["video"])
	}
	if receivedBody["caption"] != "Video caption" {
		t.Errorf("expected caption, got %v", receivedBody["caption"])
	}
	if receivedBody["disable_notification"] != true {
		t.Errorf("expected disable_notification true, got %v", receivedBody["disable_notification"])
	}
	if receivedBody["protect_content"] != true {
		t.Errorf("expected protect_content true, got %v", receivedBody["protect_content"])
	}
}

func TestSendMediaJSON_Sticker_NoCaptionOrParseMode(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":125}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, 12345, nil, "CAACAgIAAxkBAAI", "sticker", "sticker", "This should be ignored", "markdown", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["sticker"] != "CAACAgIAAxkBAAI" {
		t.Errorf("expected sticker file_id, got %v", receivedBody["sticker"])
	}
	if _, ok := receivedBody["caption"]; ok {
		t.Error("sticker should not have caption")
	}
	if _, ok := receivedBody["parse_mode"]; ok {
		t.Error("sticker should not have parse_mode")
	}
}

func TestSendMediaJSON_WithThreadID(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":126}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	threadID := int64(42)
	err := sendMediaJSON(logger, server.URL, 12345, &threadID, "https://example.com/doc.pdf", "document", "document", "Doc caption", "html", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["message_thread_id"].(float64) != 42 {
		t.Errorf("expected message_thread_id 42, got %v", receivedBody["message_thread_id"])
	}
	if receivedBody["document"] != "https://example.com/doc.pdf" {
		t.Errorf("expected document URL, got %v", receivedBody["document"])
	}
}

func TestSendMediaJSON_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: wrong file identifier/HTTP URL specified"}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, 12345, nil, "invalid_url", "photo", "photo", "", "", false, false)
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "non-OK response") {
		t.Errorf("expected non-OK response error, got: %v", err)
	}
}

func TestSendMediaMultipart_Photo(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-photo-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake jpeg data"))
	tmpFile.Close()

	var receivedFields map[string]string
	var receivedFileName string
	var receivedFileField string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("error parsing content type: %v", err)
		}
		if !strings.HasPrefix(mediaType, "multipart/") {
			t.Fatalf("expected multipart content type, got %s", mediaType)
		}

		receivedFields = make(map[string]string)
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}
			data, _ := io.ReadAll(part)
			if part.FileName() != "" {
				receivedFileName = part.FileName()
				receivedFileField = part.FormName()
			} else {
				receivedFields[part.FormName()] = string(data)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":127}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err = sendMediaMultipart(logger, server.URL, 12345, nil, tmpFile.Name(), "photo", "photo", "Upload caption", "markdown", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedFileField != "photo" {
		t.Errorf("expected file field 'photo', got %q", receivedFileField)
	}
	if receivedFileName != filepath.Base(tmpFile.Name()) {
		t.Errorf("expected filename %q, got %q", filepath.Base(tmpFile.Name()), receivedFileName)
	}
	if receivedFields["chat_id"] != "12345" {
		t.Errorf("expected chat_id '12345', got %q", receivedFields["chat_id"])
	}
	if receivedFields["caption"] != "Upload caption" {
		t.Errorf("expected caption 'Upload caption', got %q", receivedFields["caption"])
	}
	if receivedFields["parse_mode"] != "markdown" {
		t.Errorf("expected parse_mode 'markdown', got %q", receivedFields["parse_mode"])
	}
}

func TestSendMediaMultipart_WithThreadID(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-video-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake video data"))
	tmpFile.Close()

	var receivedFields map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		receivedFields = make(map[string]string)
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}
			if part.FileName() == "" {
				data, _ := io.ReadAll(part)
				receivedFields[part.FormName()] = string(data)
			} else {
				io.ReadAll(part)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":128}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	threadID := int64(99)
	err = sendMediaMultipart(logger, server.URL, -100999, &threadID, tmpFile.Name(), "video", "video", "Thread video", "html", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedFields["chat_id"] != "-100999" {
		t.Errorf("expected chat_id '-100999', got %q", receivedFields["chat_id"])
	}
	if receivedFields["message_thread_id"] != "99" {
		t.Errorf("expected message_thread_id '99', got %q", receivedFields["message_thread_id"])
	}
	if receivedFields["caption"] != "Thread video" {
		t.Errorf("expected caption 'Thread video', got %q", receivedFields["caption"])
	}
	if receivedFields["disable_notification"] != "true" {
		t.Errorf("expected disable_notification 'true', got %q", receivedFields["disable_notification"])
	}
	if receivedFields["protect_content"] != "true" {
		t.Errorf("expected protect_content 'true', got %q", receivedFields["protect_content"])
	}
}

func TestSendMediaMultipart_Sticker_NoCaptionOrParseMode(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-sticker-*.webp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake webp data"))
	tmpFile.Close()

	var receivedFields map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		receivedFields = make(map[string]string)
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}
			if part.FileName() == "" {
				data, _ := io.ReadAll(part)
				receivedFields[part.FormName()] = string(data)
			} else {
				io.ReadAll(part)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":129}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err = sendMediaMultipart(logger, server.URL, 12345, nil, tmpFile.Name(), "sticker", "sticker", "Ignored caption", "markdown", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := receivedFields["caption"]; ok {
		t.Error("sticker should not have caption field")
	}
	if _, ok := receivedFields["parse_mode"]; ok {
		t.Error("sticker should not have parse_mode field")
	}
}

func TestSendMediaMultipart_FileNotFound(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaMultipart(logger, "http://localhost", 12345, nil, "/nonexistent/file.jpg", "photo", "photo", "", "", false, false)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "error opening file") {
		t.Errorf("expected file open error, got: %v", err)
	}
}

func TestSendMediaMultipart_APIError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-error-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("data"))
	tmpFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte(`{"ok":false,"error_code":413,"description":"Request Entity Too Large"}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err = sendMediaMultipart(logger, server.URL, 12345, nil, tmpFile.Name(), "photo", "photo", "", "", false, false)
	if err == nil {
		t.Fatal("expected error for 413 response")
	}
	if !strings.Contains(err.Error(), "non-OK response") {
		t.Errorf("expected non-OK response error, got: %v", err)
	}
}

func TestCallTelegramAPI_SendMessage(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("expected path to end with /sendMessage, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":130}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	msg := TelegramMessage{
		ChatID:    12345,
		Text:      "Hello",
		ParseMode: "markdown",
	}

	err := callTelegramAPI(logger, "testtoken", "sendMessage", msg)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			t.Skip("cannot test against real Telegram API without token")
		}
	}
}

func TestSendMedia_URLDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if payload["photo"] != "https://example.com/image.jpg" {
			t.Errorf("expected photo URL, got %v", payload["photo"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":131}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, 12345, nil, "https://example.com/image.jpg", "photo", "photo", "", "", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendMediaJSON_AllMediaTypes(t *testing.T) {
	tests := []struct {
		mediaType string
		fieldName string
		media     string
	}{
		{"photo", "photo", "https://example.com/photo.jpg"},
		{"video", "video", "https://example.com/video.mp4"},
		{"audio", "audio", "https://example.com/audio.mp3"},
		{"document", "document", "https://example.com/doc.pdf"},
		{"animation", "animation", "https://example.com/anim.gif"},
		{"voice", "voice", "https://example.com/voice.ogg"},
		{"sticker", "sticker", "CAACAgIAAxkBAAI"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			var receivedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &receivedBody)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"ok":true,"result":{"message_id":200}}`))
			}))
			defer server.Close()

			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			err := sendMediaJSON(logger, server.URL, 12345, nil, tt.media, tt.fieldName, tt.mediaType, "caption", "html", false, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedBody[tt.fieldName] != tt.media {
				t.Errorf("expected %s=%q, got %v", tt.fieldName, tt.media, receivedBody[tt.fieldName])
			}

			if tt.mediaType == "sticker" {
				if _, ok := receivedBody["caption"]; ok {
					t.Error("sticker should not have caption")
				}
			} else {
				if receivedBody["caption"] != "caption" {
					t.Errorf("expected caption for %s, got %v", tt.mediaType, receivedBody["caption"])
				}
			}
		})
	}
}

func TestSendMediaJSON_EmptyCaption(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":201}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := sendMediaJSON(logger, server.URL, 12345, nil, "https://example.com/photo.jpg", "photo", "photo", "", "", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := receivedBody["caption"]; ok {
		t.Error("empty caption should not be included in payload")
	}
	if _, ok := receivedBody["parse_mode"]; ok {
		t.Error("empty parse_mode should not be included in payload")
	}
}

func TestValidMediaTypes(t *testing.T) {
	expected := map[string]string{
		"photo":     "sendPhoto",
		"video":     "sendVideo",
		"audio":     "sendAudio",
		"document":  "sendDocument",
		"animation": "sendAnimation",
		"voice":     "sendVoice",
		"sticker":   "sendSticker",
	}

	for mediaType, method := range expected {
		if validMediaTypes[mediaType] != method {
			t.Errorf("validMediaTypes[%q] = %q, want %q", mediaType, validMediaTypes[mediaType], method)
		}
	}

	invalidTypes := []string{"gif", "file", "image", "mp3", "mp4", ""}
	for _, invalid := range invalidTypes {
		if _, ok := validMediaTypes[invalid]; ok {
			t.Errorf("expected %q to be invalid media type", invalid)
		}
	}
}

func TestMediaFieldName(t *testing.T) {
	for mediaType := range validMediaTypes {
		if _, ok := mediaFieldName[mediaType]; !ok {
			t.Errorf("missing mediaFieldName entry for %q", mediaType)
		}
	}
}

func TestSendMediaMultipart_NegativeChatID(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-neg-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("data"))
	tmpFile.Close()

	var receivedFields map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		receivedFields = make(map[string]string)
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}
			if part.FileName() == "" {
				data, _ := io.ReadAll(part)
				receivedFields[part.FormName()] = string(data)
			} else {
				io.ReadAll(part)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"result":{"message_id":132}}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err = sendMediaMultipart(logger, server.URL, -1001234567890, nil, tmpFile.Name(), "photo", "photo", "", "", false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedFields["chat_id"] != "-1001234567890" {
		t.Errorf("expected chat_id '-1001234567890', got %q", receivedFields["chat_id"])
	}
}
