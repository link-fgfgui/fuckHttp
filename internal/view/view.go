package view

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	windowWidth    = 400
	settingsHeight = 500
	titleHeight    = 34
	bodyPadding    = 10
	statusHeight   = 20
	emptyHeight    = 22
	urlHeight      = 60
	buttonHeight   = 40
	gap            = 6
)

var (
	iconCopy  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/></svg>`))
	iconGear  = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1A1.7 1.7 0 0 0 9 19.4a1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1A1.7 1.7 0 0 0 4.6 9a1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/></svg>`))
	iconClose = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 4 4 20"/><path d="m4 4 16 16"/></svg>`))
)

// View draws the window from the state.
func (s *State) View(c *ui.Context) {
	theme := c.Theme()
	c.Root().Background(theme.Background)

	ui.Column(c).Fill().Children(func() {
		s.titleBar(c)
		ui.Divider(c)
		if s.Settings {
			s.settings(c)
		} else {
			s.selector(c)
		}
	})
}

func (s *State) titleBar(c *ui.Context) {
	ui.Row(c).Height(titleHeight).Padding(bodyPadding, 8).Gap(gap).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Text(c, "fuckHttp").Bold().FontSize(15).Grow(1)

		if ui.Button(c, "").Label("复制链接").Tooltip("复制链接").Size(26, 26).Padding(0).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconCopy).Size(14, 14)
		}).Clicked() {
			mygo.Clipboard.WriteText(s.URL)
			c.Toast("链接已复制到剪贴板")
		}

		if ui.Button(c, "").Label("设置").Tooltip("设置").Size(26, 26).Padding(0).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconGear).Size(14, 14)
		}).Clicked() {
			s.Settings = !s.Settings
			s.Message = ""
			s.SyncSize()
		}

		if ui.Button(c, "").Label("关闭").Tooltip("关闭").Size(26, 26).Padding(0).
			Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconClose).Size(14, 14)
		}).Clicked() {
			if s.Win != nil {
				s.Win.Close()
			}
		}
	})
}

func (s *State) selector(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Padding(bodyPadding).Gap(gap).Children(func() {
		s.statusLine(c)

		ui.Scroll(c).Height(urlHeight).Children(func() {
			ui.Text(c, s.URL).FontSize(14).Selectable()
		})

		ui.Divider(c)

		for i := range s.Browsers {
			b := &s.Browsers[i]
			if b.Hidden {
				continue
			}
			if ui.Button(c, b.Name).Height(buttonHeight).Clicked() {
				_ = b.Launch(s.URL)
				if s.Win != nil {
					s.Win.Close()
				}
			}
		}

		if s.VisibleCount() == 0 {
			ui.Text(c, "未找到浏览器").TextColor(theme.TextMuted)
		}
	})
}

func (s *State) statusLine(c *ui.Context) {
	theme := c.Theme()
	switch s.Status {
	case StatusLoading:
		ui.Row(c).Gap(gap).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Label("正在获取真实链接").Size(12, 12)
			ui.Text(c, "正在获取真实链接…").FontSize(12).TextColor(theme.Warning)
		})
	case StatusSuccess:
		if s.URL != s.Original {
			ui.Text(c, "已提取真实链接").FontSize(12).TextColor(theme.Success)
		}
	case StatusFailed:
		ui.Text(c, "提取失败: "+s.Err).FontSize(12).TextColor(theme.Danger)
	}
}

func (s *State) settings(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Padding(bodyPadding).Gap(gap).Children(func() {
		ui.Text(c, "设置").FontSize(16).Bold()
		ui.Divider(c)

		ui.Text(c, "浏览器列表").FontSize(12).TextColor(theme.TextMuted)
		changed := false
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for i := range s.Browsers {
					b := &s.Browsers[i]
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, b.Name).Grow(1).SingleLine()
						// The switch reads as "shown", which is the opposite of
						// what the config stores.
						shown := !b.Hidden
						if ui.Switch(c, &shown).Label(b.Name).Changed() {
							b.Hidden = !shown
							changed = true
						}
					})
				}
				if len(s.Browsers) == 0 {
					ui.Text(c, "未找到浏览器").TextColor(theme.TextMuted)
				}
			})
		})
		if changed {
			s.saveConfig()
		}

		ui.Divider(c)

		state := "未注册"
		if s.registered {
			state = "已注册"
		}
		ui.Text(c, "当前状态: "+state).FontSize(12).TextColor(theme.TextMuted)

		if ui.Button(c, "注册到系统").Clicked() {
			s.setRegistered(true)
		}
		if ui.Button(c, "从系统卸载").Clicked() {
			s.setRegistered(false)
		}
		if s.Message != "" {
			ui.Text(c, s.Message).FontSize(12)
		}
	})
}
