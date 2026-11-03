// Package extract turns a URL that an app intercepted into the real URL behind it.
//
// The rules mirror the original Rust implementation: QQ, WeCom and WeChat
// security pages each hide the target in a different place, and any other page
// is searched for the usual redirect parameters.
package extract

import (
	"net/url"
	"strings"
)

// Source says where a URL came from.
type Source string

const (
	SourceNone    Source = "none"
	SourceQQ      Source = "qq"
	SourceWeCom   Source = "wecom"
	SourceWeChat  Source = "wechat"
	SourceGeneric Source = "generic"
)

// Result is what Resolve found.
type Result struct {
	// URL is the real URL, or the input when nothing could be extracted.
	URL string
	// NeedAsync reports that the page must be fetched to find the URL.
	NeedAsync bool
	Source    Source
}

// redirectParams are the query parameters a redirect page may hide its target in.
var redirectParams = []string{"url", "link", "target", "redirect", "goto", "u", "q"}

// Resolve extracts the real URL from raw.
func Resolve(raw string) Result {
	u, err := url.Parse(raw)
	if err != nil {
		return Result{URL: raw, Source: SourceNone}
	}
	host := u.Hostname()

	switch {
	case strings.Contains(host, "weixin110.qq.com"):
		// The page has to be fetched: only its HTML holds the target.
		return Result{URL: raw, NeedAsync: true, Source: SourceWeChat}

	case strings.Contains(host, "open.work.weixin.qq.com"):
		if uri, ok := queryValue(u, "uri"); ok {
			decoded := percentDecode(uri)
			if !isHTTP(decoded) {
				decoded = "https://" + decoded
			}
			return Result{URL: decoded, Source: SourceWeCom}
		}

	case strings.Contains(host, "c.pc.qq.com"):
		if target, ok := queryValue(u, "url"); ok {
			if decoded := percentDecode(target); isHTTP(decoded) {
				return Result{URL: decoded, Source: SourceQQ}
			}
		}
	}

	for _, param := range redirectParams {
		target, ok := queryValue(u, param)
		if !ok {
			continue
		}
		if decoded := percentDecode(target); isHTTP(decoded) {
			return Result{URL: decoded, Source: SourceGeneric}
		}
	}

	if isHTTP(u.Fragment) {
		return Result{URL: u.Fragment, Source: SourceGeneric}
	}

	return Result{URL: raw, Source: SourceNone}
}

// queryValue returns the last value of key, as the Rust version's HashMap did.
func queryValue(u *url.URL, key string) (string, bool) {
	values := u.Query()[key]
	if len(values) == 0 {
		return "", false
	}
	return values[len(values)-1], true
}

// percentDecode decodes the escapes of an already form-decoded value, keeping
// "+" as it is, which is what urlencoding::decode did.
func percentDecode(s string) string {
	decoded, err := url.PathUnescape(s)
	if err != nil {
		return s
	}
	return decoded
}

func isHTTP(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
