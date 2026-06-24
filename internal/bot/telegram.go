package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TelegramClient struct {
	baseURL         string
	redactedBaseURL string
	client          *http.Client
	pollClient      *http.Client
}

func NewTelegramClient(token string, listenTimeout time.Duration) *TelegramClient {
	return &TelegramClient{
		baseURL:         fmt.Sprintf("https://api.telegram.org/bot%s", token),
		redactedBaseURL: "https://api.telegram.org/bot<redacted>",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		pollClient: &http.Client{
			Timeout: pollHTTPTimeout(listenTimeout),
		},
	}
}

func (c *TelegramClient) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	query := url.Values{}
	query.Set("offset", strconv.FormatInt(offset, 10))
	query.Set("timeout", strconv.Itoa(int(timeout.Seconds())))
	query.Set("allowed_updates", `["message","poll_answer"]`)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/getUpdates?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.pollClient.Do(req)
	if err != nil {
		return nil, c.sanitizeError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("getUpdates status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload UpdateResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Result, nil
}

func (c *TelegramClient) SendMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	return c.postJSON(ctx, "/sendMessage", body, nil)
}

func (c *TelegramClient) SendReply(ctx context.Context, chatID int64, replyToMessageID int64, text string) error {
	body := map[string]any{
		"chat_id":             chatID,
		"text":                text,
		"reply_to_message_id": replyToMessageID,
	}
	return c.postJSON(ctx, "/sendMessage", body, nil)
}

func (c *TelegramClient) SendPoll(ctx context.Context, chatID int64, question string, options []string) (Message, error) {
	body := map[string]any{
		"chat_id":                 chatID,
		"question":                question,
		"options":                 options,
		"is_anonymous":            false,
		"type":                    "regular",
		"allows_multiple_answers": false,
	}

	var resp SendPollResponse
	if err := c.postJSON(ctx, "/sendPoll", body, &resp); err != nil {
		return Message{}, err
	}
	return resp.Result, nil
}

func (c *TelegramClient) StopPoll(ctx context.Context, chatID, messageID int64) error {
	body := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	return c.postJSON(ctx, "/stopPoll", body, nil)
}

func (c *TelegramClient) GetChatAdministrators(ctx context.Context, chatID int64) ([]ChatMember, error) {
	body := map[string]any{
		"chat_id": chatID,
	}

	var resp ChatAdministratorsResponse
	if err := c.postJSON(ctx, "/getChatAdministrators", body, &resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *TelegramClient) SetMyCommands(ctx context.Context, commands []BotCommand, languageCode string, scope *BotCommandScope) error {
	body := map[string]any{
		"commands": commands,
	}
	if languageCode != "" {
		body["language_code"] = languageCode
	}
	if scope != nil {
		body["scope"] = scope
	}
	return c.postJSON(ctx, "/setMyCommands", body, nil)
}

func (c *TelegramClient) postJSON(ctx context.Context, endpoint string, payload any, out any) error {
	bytesBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewReader(bytesBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return c.sanitizeError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s status %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func pollHTTPTimeout(listenTimeout time.Duration) time.Duration {
	timeout := listenTimeout + 15*time.Second
	if timeout < 45*time.Second {
		return 45 * time.Second
	}
	return timeout
}

func (c *TelegramClient) sanitizeError(err error) error {
	if err == nil {
		return nil
	}

	text := strings.ReplaceAll(err.Error(), c.baseURL, c.redactedBaseURL)
	if text == err.Error() {
		return err
	}
	return errors.New(text)
}
