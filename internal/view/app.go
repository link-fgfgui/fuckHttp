// Package view holds the app's state and draws its window.
package view

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/link-fgfgui/fuckHttp/internal/browser"
	"github.com/link-fgfgui/fuckHttp/internal/config"
	"github.com/link-fgfgui/fuckHttp/internal/extract"
	"github.com/link-fgfgui/fuckHttp/internal/sysint"
)

// Status is what happened to the URL the app was given.
type Status int

const (
	// StatusLoading means the WeChat page is being read.
	StatusLoading Status = iota
	// StatusSuccess means the URL is final.
	StatusSuccess
	// StatusFailed means the page could not be read.
	StatusFailed
)

// State is everything the window shows.
type State struct {
	// Win is the window the view is drawn in, nil before it exists.
	Win *mygo.Window

	URL      string
	Original string
	Browsers []browser.Browser
	Settings bool
	Message  string
	Status   Status
	Err      string

	needAsync  bool
	registered bool
}

// blurGrace is how long a window must have had the focus before losing it
// closes the app.
const blurGrace = 500 * time.Millisecond

// New builds the state for a URL that was already resolved once.
func New(original string, result extract.Result, browsers []browser.Browser) *State {
	s := &State{
		URL:      result.URL,
		Original: original,
		Browsers: browsers,
		Status:   StatusSuccess,
	}
	if result.NeedAsync {
		s.Status = StatusLoading
		s.needAsync = true
	}
	s.registered = sysint.IsRegistered()
	return s
}

// Open creates the window and starts the work the URL needs.
func (s *State) Open() {
	s.Win = mygo.NewWindow(mygo.WindowOptions{
		Title:         "fuckHttp",
		Width:         windowWidth,
		Height:        s.Height(),
		Frameless:     true,
		Transparent:   true,
		CornerRadius:  8,
		AlwaysOnTop:   true,
		DisableResize: true,
		Content:       ui.View(s.View),
	})
	// The window can lose the focus right after it gets it, while the desktop
	// starts it; a blur a moment later is the user going elsewhere.
	var focusedAt time.Time
	s.Win.OnFocus(func() { focusedAt = time.Now() })
	s.Win.OnBlur(func() {
		if !focusedAt.IsZero() && time.Since(focusedAt) > blurGrace {
			s.Win.Close()
		}
	})
	if s.needAsync {
		go s.resolveWeChat()
	}
}

// resolveWeChat reads the security page off the main thread.
func (s *State) resolveWeChat() {
	client := &http.Client{Timeout: 10 * time.Second}
	target, err := extract.ResolveWeChat(context.Background(), client, s.Original)

	if s.Win == nil || s.Win.IsDestroyed() {
		return
	}
	s.Win.Update(func() {
		if err != nil {
			s.Status = StatusFailed
			if errors.Is(err, extract.ErrNoLink) {
				s.Err = "无法从微信页面提取链接"
			} else {
				s.Err = "网络请求失败"
			}
		} else {
			s.URL = target
			s.Status = StatusSuccess
		}
		s.SyncSize()
	})
}

// SyncSize resizes the window to what the view needs.
func (s *State) SyncSize() {
	if s.Win == nil || s.Win.IsDestroyed() {
		return
	}
	s.Win.SetSize(windowWidth, s.Height())
}

// Height is the window height the current state needs.
func (s *State) Height() int {
	if s.Settings {
		return settingsHeight
	}

	visible := s.VisibleCount()

	// The URL, the divider and either the buttons or the empty state line.
	items := 2
	if s.statusVisible() {
		items++
	}
	if visible > 0 {
		items += visible
	} else {
		items++
	}

	height := titleHeight + bodyPadding*2
	if s.statusVisible() {
		height += statusHeight
	}
	height += urlHeight + 1
	if visible > 0 {
		height += visible * buttonHeight
	} else {
		height += emptyHeight
	}
	height += (items - 1) * gap
	height += 8 // slack, so the last row never touches the edge
	return height
}

// VisibleCount is how many browser buttons show.
func (s *State) VisibleCount() int {
	n := 0
	for _, b := range s.Browsers {
		if !b.Hidden {
			n++
		}
	}
	return n
}

func (s *State) statusVisible() bool {
	switch s.Status {
	case StatusLoading, StatusFailed:
		return true
	case StatusSuccess:
		return s.URL != s.Original
	}
	return false
}

func (s *State) saveConfig() {
	_ = config.Save(config.Config{HiddenBrowsers: browser.HiddenNames(s.Browsers)})
}

// setRegistered registers or unregisters the app, asking for administrator
// rights when it has to, without blocking the view.
func (s *State) setRegistered(register bool) {
	go func() {
		var err error
		if register {
			err = sysint.EnsureRegistered()
		} else {
			err = sysint.EnsureUnregistered()
		}
		if s.Win == nil || s.Win.IsDestroyed() {
			return
		}
		s.Win.Update(func() {
			switch {
			case err != nil && register:
				s.Message = "注册失败: " + err.Error()
			case err != nil:
				s.Message = "卸载失败: " + err.Error()
			case register:
				s.Message = "注册成功!"
			default:
				s.Message = "卸载成功!"
			}
			s.registered = sysint.IsRegistered()
		})
	}()
}
