package extract

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantURL   string
		wantAsync bool
		wantSrc   Source
	}{
		{
			name:    "qq page",
			input:   "https://c.pc.qq.com/ios.html?level=14&url=https%3A%2F%2Fexample.com%2Fa%3Fb%3D1",
			wantURL: "https://example.com/a?b=1",
			wantSrc: SourceQQ,
		},
		{
			name:    "qq page encoded twice",
			input:   "https://c.pc.qq.com/ios.html?url=https%253A%252F%252Fexample.com%252Fdeep",
			wantURL: "https://example.com/deep",
			wantSrc: SourceQQ,
		},
		{
			name:    "wecom page",
			input:   "https://open.work.weixin.qq.com/wwopen/mpnews?uri=example.com%2Fpath%3Fa%3D1",
			wantURL: "https://example.com/path?a=1",
			wantSrc: SourceWeCom,
		},
		{
			name:    "wecom page without a scheme",
			input:   "https://open.work.weixin.qq.com/wwopen/mpnews?uri=example.com",
			wantURL: "https://example.com",
			wantSrc: SourceWeCom,
		},
		{
			name:    "generic url parameter",
			input:   "https://redirect.example/r?url=https%3A%2F%2Fexample.com%2Fx",
			wantURL: "https://example.com/x",
			wantSrc: SourceGeneric,
		},
		{
			name:    "generic link parameter",
			input:   "https://redirect.example/r?link=https%3A%2F%2Fexample.com%2Fy",
			wantURL: "https://example.com/y",
			wantSrc: SourceGeneric,
		},
		{
			name:    "fragment",
			input:   "https://redirect.example/#https://example.com/z",
			wantURL: "https://example.com/z",
			wantSrc: SourceGeneric,
		},
		{
			name:      "wechat page needs the network",
			input:     "https://weixin110.qq.com/security/readtemplate?t=w_security_redirect",
			wantURL:   "https://weixin110.qq.com/security/readtemplate?t=w_security_redirect",
			wantAsync: true,
			wantSrc:   SourceWeChat,
		},
		{
			name:    "non http parameter is ignored",
			input:   "https://redirect.example/r?url=javascript%3Aalert(1)",
			wantURL: "https://redirect.example/r?url=javascript%3Aalert(1)",
			wantSrc: SourceNone,
		},
		{
			name:    "plain url",
			input:   "https://example.com/plain",
			wantURL: "https://example.com/plain",
			wantSrc: SourceNone,
		},
		{
			name:    "unparsable input",
			input:   "://nonsense",
			wantURL: "://nonsense",
			wantSrc: SourceNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.input)
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
			if got.NeedAsync != tt.wantAsync {
				t.Errorf("NeedAsync = %v, want %v", got.NeedAsync, tt.wantAsync)
			}
			if got.Source != tt.wantSrc {
				t.Errorf("Source = %q, want %q", got.Source, tt.wantSrc)
			}
		})
	}
}

func TestDecodeEntities(t *testing.T) {
	input := `https&#x3a;&#x2f;&#x2f;example.com&#x2f;a&#x3f;b&#x3d;1&#x26;c&#x3d;2&amp;d=3`
	want := "https://example.com/a?b=1&c=2&d=3"
	if got := DecodeEntities(input); got != want {
		t.Errorf("DecodeEntities() = %q, want %q", got, want)
	}
}
