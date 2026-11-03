package extract

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// userAgent is a desktop browser's: the security pages answer it more readily
// than Go's default.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// ErrNoLink reports that the page was read but held no target URL.
var ErrNoLink = errors.New("no link found on the page")

var (
	reContinue = regexp.MustCompile(`"name"\s*:\s*"继续访问",\s*"url"\s*:\s*"([^"]+)"`)
	reDesc     = regexp.MustCompile(`"desc"\s*:\s*"([^"]+)"`)
	reRestore  = regexp.MustCompile(`"name"\s*:\s*"申请恢复访问",\s*"url"\s*:\s*"([^"]+)"`)
)

// ResolveWeChat fetches a WeChat security page and returns the URL it hides.
func ResolveWeChat(ctx context.Context, client *http.Client, raw string) (string, error) {
	body, err := get(ctx, client, raw)
	if err != nil {
		return "", err
	}
	html := string(body)

	// 1. The "继续访问" button.
	if m := reContinue.FindStringSubmatch(html); m != nil {
		if decoded := DecodeEntities(m[1]); isHTTP(decoded) {
			return decoded, nil
		}
	}

	// 2. The desc field of an empty page.
	if m := reDesc.FindStringSubmatch(html); m != nil {
		if decoded := DecodeEntities(m[1]); isHTTP(decoded) {
			return decoded, nil
		}
	}

	// 3. The "申请恢复访问" button: a base64 URL, or a JSON endpoint that has one.
	m := reRestore.FindStringSubmatch(html)
	if m == nil {
		return "", ErrNoLink
	}
	restore := DecodeEntities(m[1])

	if target := decodeRestoreURL(restore); target != "" {
		return target, nil
	}

	// The old format answers JSON with the button's URL.
	redirect := strings.ReplaceAll(restore, "newreadtemplate", "redirecthelpcgi")
	payload, err := get(ctx, client, redirect)
	if err != nil {
		return "", err
	}
	var response struct {
		Btns []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"btns"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return "", fmt.Errorf("unreadable restore response: %w", err)
	}
	for _, btn := range response.Btns {
		if btn.Name != "申请恢复" {
			continue
		}
		if target := decodeRestoreURL(btn.URL); target != "" {
			return target, nil
		}
	}
	return "", ErrNoLink
}

// decodeRestoreURL reads the base64 encoded target out of a restore URL.
func decodeRestoreURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	encoded, ok := queryValue(u, "url")
	if !ok {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	target := string(decoded)
	if !isHTTP(target) {
		return ""
	}
	return target
}

func get(ctx context.Context, client *http.Client, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}
