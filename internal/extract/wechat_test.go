package extract

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func page(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		default:
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestResolveWeChatContinueButton(t *testing.T) {
	body := `<script>var data = {"name": "继续访问", "url": "https://example.com/target"};</script>`
	server := page(t, body)

	got, err := ResolveWeChat(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/target" {
		t.Errorf("got %q", got)
	}
}

func TestResolveWeChatDesc(t *testing.T) {
	body := `<script>var data = {"desc": "https://example.com/from-desc"};</script>`
	server := page(t, body)

	got, err := ResolveWeChat(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/from-desc" {
		t.Errorf("got %q", got)
	}
}

func TestResolveWeChatRestoreBase64(t *testing.T) {
	target := "https://example.com/restored"
	restore := "https://weixin110.qq.com/security/newreadtemplate?url=" +
		url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(target)))
	body := `{"name": "申请恢复访问", "url": "` + restore + `"}`
	server := page(t, body)

	got, err := ResolveWeChat(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveWeChatRestoreJSON(t *testing.T) {
	target := "https://example.com/from-json"
	encoded := url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(target)))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "redirecthelpcgi") {
			_, _ = w.Write([]byte(`{"btns":[{"name":"申请恢复","url":"http://` + r.Host + `/help?url=` + encoded + `"}]}`))
			return
		}
		restore := "http://" + r.Host + "/security/newreadtemplate?url=not-base64"
		_, _ = w.Write([]byte(`{"name": "申请恢复访问", "url": "` + restore + `"}`))
	}))
	t.Cleanup(server.Close)

	got, err := ResolveWeChat(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveWeChatNoLink(t *testing.T) {
	server := page(t, "<html>nothing here</html>")

	_, err := ResolveWeChat(context.Background(), server.Client(), server.URL)
	if !errors.Is(err, ErrNoLink) {
		t.Errorf("err = %v, want ErrNoLink", err)
	}
}

func TestResolveWeChatHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	if _, err := ResolveWeChat(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("want an error")
	}
}
