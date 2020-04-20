package gwda

import (
	"encoding/base64"
	"testing"
)

func TestSession_DeviceInfo(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	dInfo, err := s.DeviceInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(dInfo)
	t.Log(dInfo.Name)
	t.Log(dInfo.UserInterfaceStyle)
}

func TestSession_BatteryInfo(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	batteryInfo, err := s.BatteryInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(batteryInfo)
	t.Log(batteryInfo.Level)
	t.Log(batteryInfo.State)
}

func TestSession_WindowSize(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	sJson, err := s.WindowSize()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sJson)
}

func TestSession_Screen(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	sJson, err := s.Screen()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sJson)
}

func TestSession_Scale(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	scale, err := s.Scale()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(scale)
}

func TestSession_StatusBarSize(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	statusBarSize, err := s.StatusBarSize()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(statusBarSize)
	t.Log(statusBarSize.Width)
	t.Log(statusBarSize.Height)
}

func TestSession_Tap(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.Tap(230, 130)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Tap(210, 290)
	if err != nil {
		t.Fatal(err)
	}
	sJson, err := s.ActiveAppInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sJson)
}

func TestSession_DoubleTap(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.DoubleTap(230, 130)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_TouchAndHold(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession("com.apple.Preferences")
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.TouchAndHold(210, 290)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_Launch(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.Launch(bundleId)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_AppTerminate(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.AppTerminate(bundleId)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_AppState(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	state, err := s.AppState(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(state)
	t.Log("app 是否前台活动中", state == WDAAppRunningFront)
}

func TestSession_SendKeys(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.SendKeys(bundleId)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_FindElement(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	// elements, err := s.FindElements("partial link text", "label=看一看")
	elements, err := s.FindElements("partial link text", "label=发现")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(elements)

	// if len(elements) == 1 {
	// 	err := elements[0].Click()
	// 	t.Log(err)
	// }
}

func TestSession_Locked(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	locked, err := s.Locked()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(locked)
}

func TestSession_Unlock(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_Lock(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.Lock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_Activate(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.Activate(bundleId)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_DeactivateApp(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.DeactivateApp(20.1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_SetPasteboardForPlaintext(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.SetPasteboardForPlaintext("abcd1234")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_SetPasteboardForImage(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.SetPasteboardForImage("/Users/hero/Documents/leixipaopao/IMG_5246.JPG")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_SetPasteboardForUrl(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	err = s.SetPasteboardForUrl("http://baidu.com")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_SetPasteboard(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	encodedContent := base64.URLEncoding.EncodeToString([]byte("https://www.google.com"))
	err = s.SetPasteboard(WDAContentTypeUrl, encodedContent)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSession_Source(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	// sTree, err := s.Source()
	sTree, err := s.Source(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sTree)
}

func TestSession_AccessibleSource(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	source, err := s.AccessibleSource()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(source)
}

func TestTmpSession(t *testing.T) {
	c, err := NewClient(deviceURL)
	if err != nil {
		t.Fatal(err)
	}
	bundleId := "com.apple.Preferences"
	s, err := c.NewSession(bundleId)
	if err != nil {
		t.Fatal(err)
	}
	Debug = true
	s.tttTmp()
	// _ = s
}
