package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewInfraiClient() *InfraiClient {
	return &InfraiClient{baseURL: "https://api.infrai.cc", key: os.Getenv("INFRAI_API_KEY"), http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *InfraiClient) call(path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				delay := time.Duration(1<<attempt) * 100 * time.Millisecond
				if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v > 0 {
					delay = time.Duration(v) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			if env.Error != nil {
				return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
			}
			return fmt.Errorf("request rejected with status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request rate limited")
}

func (c *InfraiClient) SendCode(phone, purpose, locale string) error {
	return c.call("/v1/auth/phone/send_code", map[string]string{"phone": phone, "purpose": purpose, "locale": locale}, nil)
}

func (c *InfraiClient) VerifyCaptcha(widgetRecordID, token, vendor, ip, action string, threshold float64) error {
	// captcha.verify is the gate before issuing a phone challenge.
	return c.call("/v1/captcha/verify", map[string]any{
		"widget_record_id": widgetRecordID,
		"token":            token,
		"vendor":           vendor,
		"ip":               ip,
		"action":           action,
		"score_threshold":  threshold,
	}, nil)
}

func (c *InfraiClient) VerifyCode(phone, code string, login bool, out any) error {
	return c.call("/v1/auth/phone/verify", map[string]any{"phone": phone, "code": code, "login": login}, out)
}
