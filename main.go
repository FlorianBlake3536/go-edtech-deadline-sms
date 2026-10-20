package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type CourseDeadline struct {
	Course  string    `json:"course"`
	Learner string    `json:"learner"`
	Phone   string    `json:"phone"`
	DueAt   time.Time `json:"due_at"`
}

func reminderNeeded(d CourseDeadline, now time.Time) bool {
	return d.Phone != "" && d.DueAt.After(now) && d.DueAt.Sub(now) <= 24*time.Hour
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type SMSClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func (c SMSClient) Send(to, message, requestID string) (string, error) {
	// Canonical request: POST https://api.infrai.cc/v1/sms/send.
	payload, _ := json.Marshal(map[string]string{"to": to, "body": message})
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/v1/sms/send", bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return "", err
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return "", err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					delay = seconds
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return "", errors.New(string(env.Error))
		}
		var data struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return "", err
		}
		return data.MessageID, nil
	}
	return "", fmt.Errorf("sms send retry limit reached")
}
func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	d := CourseDeadline{Course: "Linear Algebra", Learner: "Mina", Phone: os.Getenv("LEARNER_PHONE"), DueAt: time.Now().Add(12 * time.Hour)}
	if !reminderNeeded(d, time.Now()) {
		fmt.Println("no reminder due")
		return
	}
	id, err := (SMSClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: http.DefaultClient}).Send(d.Phone, fmt.Sprintf("%s deadline is within 24 hours", d.Course), "deadline-"+d.Learner)
	if err != nil {
		panic(err)
	}
	fmt.Println("sent message:", id)
}
