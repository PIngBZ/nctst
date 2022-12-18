package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math/rand"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/PIngBZ/nctst"
	"github.com/PIngBZ/nctst/client/core"
	"github.com/PIngBZ/tun2socks/v2/engine"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	config *core.Config

	App        fyne.App
	MainWindow fyne.Window
)

type FontTheme struct{}

func (*FontTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return theme.DefaultTheme().Font(s)
	}
	if s.Bold {
		if s.Italic {
			return theme.DefaultTheme().Font(s)
		}
		return resourceConsolasYaheiTtf
	}
	if s.Italic {
		return theme.DefaultTheme().Font(s)
	}
	return resourceConsolasYaheiTtf
}

func (*FontTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (*FontTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (*FontTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}

func init() {
	rand.Seed(time.Now().Unix())
	nctst.OpenLog()

	var err error
	config, err = core.ParseConfig("config")
	nctst.CheckError(err)

	nctst.CommandXorKey = config.Key
}

func main() {
	initMainWindow()

	engine.Stop()
}

func initMainWindow() {
	App = app.New()
	App.Settings().SetTheme(&FontTheme{})

	MainWindow = App.NewWindow("nctst GUI for windows")
	MainWindow.Resize(fyne.NewSize(600, 400))
	MainWindow.SetFixedSize(true)

	MainWindow.SetMaster()

	if desk, ok := App.(desktop.App); ok {
		mainWindowVisible := true

		var showHide = "显示窗口"
		if !mainWindowVisible {
			showHide = "隐藏窗口"
		}

		item := fyne.NewMenuItem(showHide, nil)
		menu := fyne.NewMenu("nctst", item)

		item.Action = func() {
			if mainWindowVisible {
				MainWindow.Hide()
				mainWindowVisible = false
				item.Label = "显示窗口"
			} else {
				MainWindow.Show()
				mainWindowVisible = true
				item.Label = "隐藏窗口"
			}
			menu.Refresh()
		}
		desk.SetSystemTrayMenu(menu)

		MainWindow.SetCloseIntercept(func() {
			MainWindow.Hide()
			mainWindowVisible = false
			item.Label = "显示窗口"
			menu.Refresh()
		})
	}

	infoView := widget.NewRichTextWithText("启动")

	MainWindow.SetContent(container.NewMax(infoView))

	go showInputCode(func(code int) {
		observer := make(chan *core.ClientStatus, 8)
		core.AttachStatusObserver(observer)
		go daemon(observer, infoView)

		go func() {
			if !startProxy(code) {
				return
			}
			addInfoLine(infoView, "\n\n创建虚拟网卡...")
			startTunDevice()
			showSuccessInfo(infoView)
		}()
	})

	MainWindow.ShowAndRun()
}

func showInputCode(success func(int)) {
	entry := widget.NewEntry()

	items := []*widget.FormItem{
		{Text: "Code", Widget: entry}}

	dialog.NewForm("输入验证码", "确定", "取消", items, func(b bool) {
		if b {
			code, err := strconv.Atoi(entry.Text)
			if err == nil && code < 1000 || code > 9999 {
				err = fmt.Errorf("error code %s", entry.Text)
			}

			if err != nil {
				showErrorDlg(err, func() {
					App.Quit()
				})
			} else {
				if err := checkCode(code); err != nil {
					showErrorDlg(err, func() {
						App.Quit()
					})
				} else {
					success(code)
				}
			}
		} else {
			App.Quit()
		}
	}, MainWindow).Show()
}

func checkCode(code int) error {
	client := retryablehttp.NewClient()
	client.HTTPClient.Timeout = time.Second * 15
	client.RetryMax = 3

	url := "http://" + config.Manager.Address() + "/checkcode?code=" + strconv.Itoa(code)
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(config.UserName, config.PassWord)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("error response code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respObj nctst.APIResponse
	err = json.Unmarshal(data, &respObj)
	if err != nil {
		return err
	}

	if respObj.Code != nctst.APIResponseCode_Success {
		return fmt.Errorf("error response obj code: %d", respObj.Code)
	}

	return nil
}

func showErrorDlg(err error, onClosed func()) {
	errdlg := dialog.NewError(err, MainWindow)
	errdlg.SetOnClosed(func() {
		if onClosed != nil {
			onClosed()
		}
	})
	errdlg.Show()
}

func startProxy(code int) bool {
	if err := core.Start(config, code); err != nil {
		showErrorDlg(fmt.Errorf("core.Start %+v", err), func() {
			App.Quit()
		})
		return false
	}
	return true
}

func startTunDevice() {
	key := new(engine.Key)
	key.Device = fmt.Sprintf("tun://wintun?ip=%s&route=%s", config.TunIP, config.TunRoute)
	if config.Listen[0] == ':' {
		key.Proxy = "socks5://127.0.0.1" + config.Listen
	} else {
		key.Proxy = "socks5://" + config.Listen
	}
	key.LogLevel = "info"

	engine.Insert(key)

	if err := engine.Start(); err != nil {
		showErrorDlg(err, func() {
			App.Quit()
		})
	}
}

func daemon(observer chan *core.ClientStatus, text *widget.RichText) {
	last := core.ClientStatusStep_Init
	addStatusText(last, text)

	for status := range observer {
		if status.GetStat() != last {
			last = status.GetStat()
			addStatusText(status.GetStat(), text)
		}
	}
}

func addStatusText(status core.ClientStatusStep, text *widget.RichText) {
	txt := "\n"

	switch status {
	case core.ClientStatusStep_Init:
		txt += "初始化完成..."
	case core.ClientStatusStep_GetProxyList:
		txt += "获取代理列表并测速选择..."
	case core.ClientStatusStep_Login:
		txt += "登录中..."
	case core.ClientStatusStep_StartUpstream:
		txt += "连接代理服务器..."
	case core.ClientStatusStep_StartMapLocal:
		txt += "端口映射..."
	case core.ClientStatusStep_StartLocalService:
		txt += "本地listen..."
	case core.ClientStatusStep_CheckingConnection:
		txt += "连接目标服务器..."
	case core.ClientStatusStep_Running:
		txt += "连接成功~~"
	case core.ClientStatusStep_Failed:
		txt += "连接失败！！！"
	}
	addInfoLine(text, txt)
}

func showSuccessInfo(text *widget.RichText) {
	addInfoLine(text, "\n\n完成\n\n")
	n := fmt.Sprintf("延迟： %d\n本地socks5： %s\n虚拟网卡地址： %s\n自动拦截并转发请求网段： %s",
		core.Status.GetPing(),
		config.Listen,
		config.TunIP,
		config.TunRoute)
	addInfoLine(text, n)
}

func addInfoLine(text *widget.RichText, str string) {
	text.Segments = append(text.Segments, &widget.TextSegment{Style: widget.RichTextStyleInline, Text: str})
	text.Resize(text.MinSize())
}
