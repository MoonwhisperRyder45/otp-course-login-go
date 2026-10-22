package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReportCountsVerifiedLearners(t *testing.T) {
	a := &app{learners: map[string]learner{"+15550001": {Phone: "+15550001"}}}
	req := httptest.NewRequest("GET", "/report/deadlines", strings.NewReader(""))
	rec := httptest.NewRecorder()
	a.report(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"learners":1`) {
		t.Fatalf("report = %d %s", rec.Code, rec.Body.String())
	}
}

func TestVerifyCaptchaIncludesWidgetRecordID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/captcha/verify" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["widget_record_id"] != "widget-123" || body["token"] != "token-123" {
			t.Errorf("captcha request = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	c := &InfraiClient{baseURL: server.URL, http: server.Client()}
	if err := c.VerifyCaptcha("widget-123", "token-123", "default", "127.0.0.1", "phone_login", 0.5); err != nil {
		t.Fatal(err)
	}
}
