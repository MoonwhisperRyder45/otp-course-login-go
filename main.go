package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type learner struct {
	Phone    string    `json:"phone"`
	JoinedAt time.Time `json:"joined_at"`
}
type app struct {
	client   *InfraiClient
	mu       sync.RWMutex
	learners map[string]learner
}

func (a *app) requestCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Phone          string `json:"phone"`
		CaptchaToken   string `json:"captcha_token"`
		WidgetRecordID string `json:"widget_record_id"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Phone == "" || in.CaptchaToken == "" || in.WidgetRecordID == "" {
		http.Error(w, "phone, captcha_token and widget_record_id are required", http.StatusBadRequest)
		return
	}
	if err := a.client.VerifyCaptcha(in.WidgetRecordID, in.CaptchaToken, "default", r.RemoteAddr, "phone_login", 0.5); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.client.SendCode(in.Phone, "login", "en-US"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "code_sent"})
}

func (a *app) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct{ Phone, Code string }
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Phone == "" || in.Code == "" {
		http.Error(w, "phone and code are required", http.StatusBadRequest)
		return
	}
	var data map[string]any
	if err := a.client.VerifyCode(in.Phone, in.Code, true, &data); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	a.mu.Lock()
	a.learners[in.Phone] = learner{Phone: in.Phone, JoinedAt: time.Now().UTC()}
	a.mu.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"status": "logged_in"})
}

func (a *app) report(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]any{"course": "Algebra 1", "deadline": "2026-10-01", "learners": len(a.learners)})
}

func main() {
	a := &app{client: NewInfraiClient(), learners: map[string]learner{}}
	http.HandleFunc("/login/request", a.requestCode)
	http.HandleFunc("/login/verify", a.verify)
	http.HandleFunc("/report/deadlines", a.report)
	http.ListenAndServe(":8080", nil)
}
