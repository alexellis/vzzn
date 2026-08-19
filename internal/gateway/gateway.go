// Package gateway talks to the toilgate OpenAI-compatible inference API.
package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Part is one content part of a chat message.
type Part struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL carries an image as a data URL.
type ImageURL struct {
	URL string `json:"url"`
}

// Message is a chat message with multimodal content.
type Message struct {
	Role    string `json:"role"`
	Content []Part `json:"content"`
}

// Request is a chat-completions request.
type Request struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	Stream          bool      `json:"stream"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
}

// backoffs bounds the transient-retry schedule. The worst case (every
// attempt failing) lands its last attempt at ~105s, inside a default
// timeout of two minutes, so backend warm-up (~80s) self-heals while
// unknown-model errors are bounded rather than open-ended.
var backoffs = []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second}

// Served reports whether model is present in the gateway's live catalogue
// (GET <origin>/models.json, open — no auth). It is how an unknown model is
// told apart from a transient backend reload before any request is sent.
func Served(baseURL, model string) (bool, error) {
	origin := strings.TrimSuffix(strings.TrimSuffix(baseURL, "/"), "/v1")
	url := origin + "/models.json"
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(url)
	if err != nil {
		return false, fmt.Errorf("catalogue %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return false, fmt.Errorf("catalogue %s: %s: %s", url, resp.Status, strings.TrimSpace(string(b)))
	}
	var doc struct {
		Models []string `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return false, fmt.Errorf("catalogue %s: %w", url, err)
	}
	for _, id := range doc.Models {
		if id == model {
			return true, nil
		}
	}
	return false, nil
}

// Complete streams a chat completion to out. The model is validated against
// the live catalogue first, so an unknown model fails fast (one round-trip)
// rather than burning a retry schedule. When the model is validated-present
// but a request still transiently 404s ("not served" during a backend
// reload), it retries with backoff within the overall timeout. Progress
// notices go to progress; answer content goes to out.
func Complete(baseURL, token string, req Request, out, progress io.Writer, timeout time.Duration) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	served, cerr := Served(baseURL, req.Model)
	if cerr != nil {
		return cerr
	}
	if !served {
		return fmt.Errorf("model %q not served by toilgate (unknown, or backend restarting)", req.Model)
	}

	deadline := time.Now().Add(timeout)
	var lastErr error
	for attempt := 0; ; attempt++ {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timed out after %d attempt(s) within %v (last: %v)", attempt, timeout, lastErr)
		}
		lastErr = doOnce(&http.Client{Timeout: remaining}, baseURL, token, body, req.Stream, out, progress)
		if lastErr == nil {
			return nil
		}
		if !isTransient(lastErr) {
			return lastErr
		}
		if attempt >= len(backoffs) || time.Until(deadline) <= backoffs[attempt] {
			return fmt.Errorf("timed out after %d attempt(s) within %v", attempt+1, timeout)
		}
		d := backoffs[attempt]
		fmt.Fprintf(progress, "toilgate transient (attempt %d/%d), retrying in %v...\n", attempt+1, len(backoffs)+1, d)
		time.Sleep(d)
	}
}

type transientErr struct{ msg string }

func (e transientErr) Error() string { return e.msg }

func isTransient(err error) bool {
	_, ok := err.(transientErr)
	return ok
}

func doOnce(client *http.Client, baseURL, token string, body []byte, stream bool, out, progress io.Writer) error {
	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return transientErr{msg: fmt.Sprintf("request: %v", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if strings.Contains(string(b), "not served") {
			return transientErr{msg: "backend mid-reload (404 not served)"}
		}
		return fmt.Errorf("http 404: %s", strings.TrimSpace(string(b)))
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("http %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if stream {
		return streamSSE(resp.Body, out)
	}
	return streamJSON(resp.Body, out)
}

// streamSSE parses a streamed chat-completion response, writing each
// content delta to out as it arrives.
func streamSSE(r io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content != "" {
				if _, err := io.WriteString(out, c.Delta.Content); err != nil {
					return err
				}
			}
		}
	}
	return scanner.Err()
}

// streamJSON parses a non-streamed chat-completion response and writes the
// single completion's content to out.
func streamJSON(r io.Reader, out io.Writer) error {
	var doc struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return fmt.Errorf("parsing completion: %w", err)
	}
	if len(doc.Choices) == 0 {
		return fmt.Errorf("completion returned no choices")
	}
	_, err := io.WriteString(out, doc.Choices[0].Message.Content)
	return err
}
