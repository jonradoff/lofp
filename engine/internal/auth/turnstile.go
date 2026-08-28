package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// VerifyCaptcha checks a Cloudflare Turnstile response token against Cloudflare's
// siteverify API to confirm the registration request came from a real browser
// session, not a script. If no turnstile secret is configured (e.g. local dev),
// verification is skipped entirely so registration keeps working without the
// widget — mirrors how Google login silently disables itself with no client ID.
func (s *Service) VerifyCaptcha(ctx context.Context, token, remoteIP string) error {
	if s.turnstileSecret == "" {
		return nil
	}
	if token == "" {
		return fmt.Errorf("please complete the verification challenge")
	}

	form := url.Values{
		"secret":   {s.turnstileSecret},
		"response": {token},
	}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("captcha verification unavailable, please try again")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("captcha verification unavailable, please try again")
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || !result.Success {
		return fmt.Errorf("captcha verification failed, please try again")
	}
	return nil
}
