package whatsapp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	Path       string
	HTTPClient *http.Client
}

type SendMessageRequest struct {
	Phone        string `json:"phone"`
	Message      string `json:"message"`
	IsForwarded  bool   `json:"is_forwarded"`
	Duration     int    `json:"duration"`
}

type SendMessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
	} `json:"data"`
}

type WebhookMessage struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
	From    string `json:"from"`
	To      string `json:"to"`
	Time    string `json:"time"`
}

func NewClient(baseURL, username, password, path string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Password: password,
		Path:     path,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Convert phone number from 08xxx to 628xxx format
func (c *Client) convertPhoneNumber(phone string) string {
	if strings.HasPrefix(phone, "08") {
		return "628" + phone[2:]
	}
	return phone
}

// Send message via WhatsApp
func (c *Client) SendMessage(phone, message string, isForwarded bool, duration int) (*SendMessageResponse, error) {
	// Convert phone number format
	convertedPhone := c.convertPhoneNumber(phone)
	
	// Prepare request data
	requestData := SendMessageRequest{
		Phone:       convertedPhone + "@s.whatsapp.net",
		Message:     message,
		IsForwarded: isForwarded,
		Duration:    duration,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create request URL
	url := fmt.Sprintf("%s/%s/send/message", c.BaseURL, c.Path)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	
	// Create Basic Auth token
	auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	// Send request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response SendMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Send simple text message
func (c *Client) SendTextMessage(phone, message string) error {
	_, err := c.SendMessage(phone, message, false, 0)
	return err
}

// Send message with forwarding
func (c *Client) SendForwardedMessage(phone, message string, duration int) error {
	_, err := c.SendMessage(phone, message, true, duration)
	return err
}
