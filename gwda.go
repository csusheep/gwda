package gwda

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HTTPClient The default client to use to communicate with the WebDriver server.
var HTTPClient = http.DefaultClient

var (
	DefaultWaitTimeout  = 60 * time.Second
	DefaultWaitInterval = 400 * time.Millisecond

	DefaultKeepAliveInterval = 30 * time.Second
)

func newRequest(method string, url string, rawBody []byte) (request *http.Request, err error) {
	var header = map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
		"Accept":       "application/json",
	}
	if request, err = http.NewRequest(method, url, bytes.NewBuffer(rawBody)); err != nil {
		return nil, err
	}
	for k, v := range header {
		request.Header.Set(k, v)
	}
	return
}

var _mUSB sync.Mutex

func executeHTTP(method string, rawURL string, rawBody []byte, usbHTTPClient ...*http.Client) (rawResp rawResponse, err error) {
	debugLog(fmt.Sprintf("--> %s %s\n%s", method, rawURL, rawBody))
	var req *http.Request
	if req, err = newRequest(method, rawURL, rawBody); err != nil {
		return
	}

	tmpHTTPClient := HTTPClient
	if len(usbHTTPClient) != 0 {
		tmpHTTPClient = usbHTTPClient[0]
		_mUSB.Lock()
		defer _mUSB.Unlock()
	}

	start := time.Now()
	var resp *http.Response
	if resp, err = tmpHTTPClient.Do(req); err != nil {
		return nil, err
	}
	defer func() {
		// https://github.com/etcd-io/etcd/blob/v3.3.25/pkg/httputil/httputil.go#L16-L22
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	rawResp, err = ioutil.ReadAll(resp.Body)
	debugLog(fmt.Sprintf("<-- %s %s %d %s\n%s\n", method, rawURL, resp.StatusCode, time.Since(start), rawResp))
	if err != nil {
		return nil, err
	}

	if err = rawResp.checkErr(); err != nil {
		if resp.StatusCode == http.StatusOK {
			return rawResp, nil
		}
		return nil, err
	}

	return
}

func keepAlive(d WebDriver) {
	go func() {
		if DefaultKeepAliveInterval <= 0 {
			return
		}
		ticker := time.NewTicker(DefaultKeepAliveInterval)
		for {
			select {
			case <-ticker.C:
				if _, err := d.Status(); err != nil {
					ticker.Stop()
					return
				}
			}
		}
	}()
}

var debugFlag = false

// SetDebug sets debug mode
func SetDebug(debug bool) {
	debugFlag = debug
}

func debugLog(msg string) {
	if !debugFlag {
		return
	}
	log.Println("[GWDA-DEBUG] " + msg)
}

func convertToHTTPClient(_conn net.Conn) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return _conn, nil
			},
		},
	}
}

type rawResponse []byte

func (r rawResponse) checkErr() (err error) {
	var reply = new(struct {
		Value struct {
			Err       string `json:"error"`
			Message   string `json:"message"`
			Traceback string `json:"traceback"`
		}
	})
	if err = json.Unmarshal(r, reply); err != nil {
		return err
	}
	if reply.Value.Err != "" {
		errText := reply.Value.Message
		re := regexp.MustCompile(`{.+?=(.+?)}`)
		if re.MatchString(reply.Value.Message) {
			subMatch := re.FindStringSubmatch(reply.Value.Message)
			errText = subMatch[len(subMatch)-1]
		}
		return fmt.Errorf("%s: %s", reply.Value.Err, errText)
	}
	return
}

func (r rawResponse) valueConvertToString() (s string, err error) {
	var reply = new(struct{ Value string })
	if err = json.Unmarshal(r, reply); err != nil {
		return "", err
	}
	s = reply.Value
	return
}

func (r rawResponse) valueConvertToBool() (b bool, err error) {
	var reply = new(struct{ Value bool })
	if err = json.Unmarshal(r, reply); err != nil {
		return false, err
	}
	b = reply.Value
	return
}

func (r rawResponse) valueConvertToSessionInfo() (sessionInfo SessionInfo, err error) {
	var reply = new(struct{ Value struct{ SessionInfo } })
	if err = json.Unmarshal(r, reply); err != nil {
		return SessionInfo{}, err
	}
	sessionInfo = reply.Value.SessionInfo
	return
}

func (r rawResponse) valueConvertToJsonRawMessage() (raw json.RawMessage, err error) {
	var reply = new(struct{ Value json.RawMessage })
	if err = json.Unmarshal(r, reply); err != nil {
		return nil, err
	}
	raw = reply.Value
	return
}

func (r rawResponse) valueDecodeAsBase64() (raw *bytes.Buffer, err error) {
	var str string
	if str, err = r.valueConvertToString(); err != nil {
		return nil, err
	}
	var decodeString []byte
	if decodeString, err = base64.StdEncoding.DecodeString(str); err != nil {
		return nil, err
	}
	raw = bytes.NewBuffer(decodeString)
	return
}

var errNoSuchElement = errors.New("no such element")

func (r rawResponse) valueConvertToElementID() (id string, err error) {
	var reply = new(struct{ Value map[string]string })
	if err = json.Unmarshal(r, reply); err != nil {
		return "", err
	}
	if len(reply.Value) == 0 {
		return "", errNoSuchElement
	}
	if id = elementIDFromValue(reply.Value); id == "" {
		return "", fmt.Errorf("invalid element returned: %+v", reply)
	}
	return
}

func (r rawResponse) valueConvertToElementIDs() (IDs []string, err error) {
	var reply = new(struct{ Value []map[string]string })
	if err = json.Unmarshal(r, reply); err != nil {
		return nil, err
	}
	if len(reply.Value) == 0 {
		return nil, errNoSuchElement
	}
	IDs = make([]string, len(reply.Value))
	for i, elem := range reply.Value {
		var id string
		if id = elementIDFromValue(elem); id == "" {
			return nil, fmt.Errorf("invalid element returned: %+v", reply)
		}
		IDs[i] = id
	}
	return
}

type AlertAction string

const (
	AlertActionAccept  AlertAction = "accept"
	AlertActionDismiss AlertAction = "dismiss"
)

type Capabilities map[string]interface{}

func NewCapabilities() Capabilities {
	return make(Capabilities)
}

func (caps Capabilities) WithAppLaunchOption(launchOpt AppLaunchOption) Capabilities {
	for k, v := range launchOpt {
		caps[k] = v
	}
	return caps
}

// WithDefaultAlertAction
func (caps Capabilities) WithDefaultAlertAction(alertAction AlertAction) Capabilities {
	caps["defaultAlertAction"] = alertAction
	return caps
}

// WithMaxTypingFrequency
//
//	Defaults to `60`.
func (caps Capabilities) WithMaxTypingFrequency(n int) Capabilities {
	if n <= 0 {
		n = 60
	}
	caps["maxTypingFrequency"] = n
	return caps
}

// WithWaitForIdleTimeout
//
//	Defaults to `10`
func (caps Capabilities) WithWaitForIdleTimeout(second float64) Capabilities {
	caps["waitForIdleTimeout"] = second
	return caps
}

// WithShouldUseTestManagerForVisibilityDetection If set to YES will ask TestManagerDaemon for element visibility
//
//	Defaults to  `false`
func (caps Capabilities) WithShouldUseTestManagerForVisibilityDetection(b bool) Capabilities {
	caps["shouldUseTestManagerForVisibilityDetection"] = b
	return caps
}

// WithShouldUseCompactResponses If set to YES will use compact (standards-compliant) & faster responses
//
//	Defaults to `true`
func (caps Capabilities) WithShouldUseCompactResponses(b bool) Capabilities {
	caps["shouldUseCompactResponses"] = b
	return caps
}

// WithElementResponseAttributes If shouldUseCompactResponses == NO,
// is the comma-separated list of fields to return with each element.
//
//	Defaults to `type,label`.
func (caps Capabilities) WithElementResponseAttributes(s string) Capabilities {
	caps["elementResponseAttributes"] = s
	return caps
}

// WithShouldUseSingletonTestManager
//
//	Defaults to `true`
func (caps Capabilities) WithShouldUseSingletonTestManager(b bool) Capabilities {
	caps["shouldUseSingletonTestManager"] = b
	return caps
}

// WithDisableAutomaticScreenshots
//
//	Defaults to `true`
func (caps Capabilities) WithDisableAutomaticScreenshots(b bool) Capabilities {
	caps["disableAutomaticScreenshots"] = b
	return caps
}

// WithShouldTerminateApp
//
//	Defaults to `true`
func (caps Capabilities) WithShouldTerminateApp(b bool) Capabilities {
	caps["shouldTerminateApp"] = b
	return caps
}

// WithEventloopIdleDelaySec
// Delays the invocation of '-[XCUIApplicationProcess setEventLoopHasIdled:]' by the timer interval passed.
// which is skipped on setting it to zero.
func (caps Capabilities) WithEventloopIdleDelaySec(second float64) Capabilities {
	caps["eventloopIdleDelaySec"] = second
	return caps
}

type SessionInfo struct {
	SessionId    string `json:"sessionId"`
	Capabilities struct {
		Device             string `json:"device"`
		BrowserName        string `json:"browserName"`
		SdkVersion         string `json:"sdkVersion"`
		CFBundleIdentifier string `json:"CFBundleIdentifier"`
	} `json:"capabilities"`
}

type DeviceStatus struct {
	Message string `json:"message"`
	State   string `json:"state"`
	OS      struct {
		TestmanagerdVersion int    `json:"testmanagerdVersion"`
		Name                string `json:"name"`
		SdkVersion          string `json:"sdkVersion"`
		Version             string `json:"version"`
	} `json:"os"`
	IOS struct {
		IP               string `json:"ip"`
		SimulatorVersion string `json:"simulatorVersion"`
	} `json:"ios"`
	Ready bool `json:"ready"`
	Build struct {
		Time                    string `json:"time"`
		ProductBundleIdentifier string `json:"productBundleIdentifier"`
	} `json:"build"`
}

type DeviceInfo struct {
	TimeZone           string `json:"timeZone"`
	CurrentLocale      string `json:"currentLocale"`
	Model              string `json:"model"`
	UUID               string `json:"uuid"`
	UserInterfaceIdiom int    `json:"userInterfaceIdiom"`
	UserInterfaceStyle string `json:"userInterfaceStyle"`
	Name               string `json:"name"`
	IsSimulator        bool   `json:"isSimulator"`
}

type Location struct {
	AuthorizationStatus int     `json:"authorizationStatus"`
	Longitude           float64 `json:"longitude"`
	Latitude            float64 `json:"latitude"`
	Altitude            float64 `json:"altitude"`
}

type BatteryInfo struct {
	// Battery level in range [0.0, 1.0], where 1.0 means 100% charge.
	Level float64 `json:"level"`

	// Battery state ( 1: on battery, discharging; 2: plugged in, less than 100%, 3: plugged in, at 100% )
	State BatteryState `json:"state"`
}

type BatteryState int

const (
	_                                  = iota
	BatteryStateUnplugged BatteryState = iota // on battery, discharging
	BatteryStateCharging                      // plugged in, less than 100%
	BatteryStateFull                          // plugged in, at 100%
)

func (v BatteryState) String() string {
	switch v {
	case BatteryStateUnplugged:
		return "On battery, discharging"
	case BatteryStateCharging:
		return "Plugged in, less than 100%"
	case BatteryStateFull:
		return "Plugged in, at 100%"
	default:
		return "UNKNOWN"
	}
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Screen struct {
	StatusBarSize Size    `json:"statusBarSize"`
	Scale         float64 `json:"scale"`
}

type AppInfo struct {
	ProcessArguments struct {
		Env  interface{}   `json:"env"`
		Args []interface{} `json:"args"`
	} `json:"processArguments"`
	Name string `json:"name"`
	AppBaseInfo
}

type AppBaseInfo struct {
	Pid      int    `json:"pid"`
	BundleId string `json:"bundleId"`
}

type AppState int

const (
	AppStateNotUnkown    AppState = 0
	AppStateNotRunning   AppState = 1
	AppStateRunningBack  AppState = 3
	AppStateRunningFront AppState = 4
)

func (v AppState) String() string {
	switch v {
	case AppStateNotRunning:
		return "Not Running"
	case AppStateRunningBack:
		return "Running (Back)"
	case AppStateRunningFront:
		return "Running (Front)"
	default:
		return "UNKNOWN"
	}
}

// AppLaunchOption Configure app launch parameters
type AppLaunchOption map[string]interface{}

func NewAppLaunchOption() AppLaunchOption {
	return make(AppLaunchOption)
}
func (opt AppLaunchOption) WithBundleId(bundleId string) AppLaunchOption {
	opt["bundleId"] = bundleId
	return opt
}

// WithShouldWaitForQuiescence whether to wait for quiescence on application startup
//
//	Defaults to `true`
func (opt AppLaunchOption) WithShouldWaitForQuiescence(b bool) AppLaunchOption {
	opt["shouldWaitForQuiescence"] = b
	return opt
}

// WithArguments The optional array of application command line arguments.
// The arguments are going to be applied if the application was not running before.
func (opt AppLaunchOption) WithArguments(args []string) AppLaunchOption {
	opt["arguments"] = args
	return opt
}

// WithEnvironment The optional dictionary of environment variables for the application, which is going to be executed.
// The environment variables are going to be applied if the application was not running before.
func (opt AppLaunchOption) WithEnvironment(env map[string]string) AppLaunchOption {
	opt["environment"] = env
	return opt
}

// PasteboardType The type of the item on the pasteboard.
type PasteboardType string

const (
	PasteboardTypePlaintext PasteboardType = "plaintext"
	PasteboardTypeImage     PasteboardType = "image"
	PasteboardTypeUrl       PasteboardType = "url"
)

const (
	TextBackspace string = "\u0008"
	TextDelete    string = "\u007F"
)

// type KeyboardKeyLabel string
//
// const (
// 	KeyboardKeyReturn = "return"
// )

// DeviceButton A physical button on an iOS device.
type DeviceButton string

const (
	DeviceButtonHome       DeviceButton = "home"
	DeviceButtonVolumeUp   DeviceButton = "volumeUp"
	DeviceButtonVolumeDown DeviceButton = "volumeDown"
)

type NotificationType string

const (
	NotificationTypePlain  NotificationType = "plain"
	NotificationTypeDarwin NotificationType = "darwin"
)

// EventPageID The event page identifier
type EventPageID int

const EventPageIDConsumer EventPageID = 0x0C

// EventUsageID The event usage identifier (usages are defined per-page)
type EventUsageID int

const (
	EventUsageIDCsmrVolumeUp   EventUsageID = 0xE9
	EventUsageIDCsmrVolumeDown EventUsageID = 0xEA
	EventUsageIDCsmrHome       EventUsageID = 0x40
	EventUsageIDCsmrPower      EventUsageID = 0x30
	EventUsageIDCsmrSnapshot   EventUsageID = 0x65 // Power + Home
)

type Orientation string

const (
	// OrientationPortrait Device oriented vertically, home button on the bottom
	OrientationPortrait Orientation = "PORTRAIT"

	// OrientationPortraitUpsideDown Device oriented vertically, home button on the top
	OrientationPortraitUpsideDown Orientation = "UIA_DEVICE_ORIENTATION_PORTRAIT_UPSIDEDOWN"

	// OrientationLandscapeLeft Device oriented horizontally, home button on the right
	OrientationLandscapeLeft Orientation = "LANDSCAPE"

	// OrientationLandscapeRight Device oriented horizontally, home button on the left
	OrientationLandscapeRight Orientation = "UIA_DEVICE_ORIENTATION_LANDSCAPERIGHT"
)

type Rotation struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

// SourceOption Configure the format or attribute of the Source
type SourceOption map[string]interface{}

func NewSourceOption() SourceOption {
	return make(SourceOption)
}

// WithFormatAsJson Application elements tree in form of json string
func (opt SourceOption) WithFormatAsJson() SourceOption {
	opt["format"] = "json"
	return opt
}

// WithFormatAsXml Application elements tree in form of xml string
func (opt SourceOption) WithFormatAsXml() SourceOption {
	opt["format"] = "xml"
	return opt
}

// WithFormatAsDescription Application elements tree in form of internal XCTest debugDescription string
func (opt SourceOption) WithFormatAsDescription() SourceOption {
	opt["format"] = "description"
	return opt
}

// WithExcludedAttributes Excludes the given attribute names.
// only `xml` is supported.
func (opt SourceOption) WithExcludedAttributes(attributes []string) SourceOption {
	if vFormat, ok := opt["format"]; ok && vFormat != "xml" {
		return opt
	}
	opt["excluded_attributes"] = strings.Join(attributes, ",")
	return opt
}

const (
	// legacyWebElementIdentifier is the string constant used in the old
	// WebDriver JSON protocol that is the key for the map that contains an
	// unique element identifier.
	legacyWebElementIdentifier = "ELEMENT"

	// webElementIdentifier is the string constant defined by the W3C
	// specification that is the key for the map that contains a unique element identifier.
	webElementIdentifier = "element-6066-11e4-a52e-4f735466cecf"
)

func elementIDFromValue(val map[string]string) string {
	for _, key := range []string{webElementIdentifier, legacyWebElementIdentifier} {
		if v, ok := val[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

type BySelector struct {
	ClassName ElementType `json:"class name"`

	// isSearchByIdentifier
	Name            string `json:"name"`
	Id              string `json:"id"`
	AccessibilityId string `json:"accessibility id"`
	// isSearchByIdentifier

	// partialSearch
	LinkText        ElementAttribute `json:"link text"`
	PartialLinkText ElementAttribute `json:"partial link text"`
	// partialSearch

	Predicate string `json:"predicate string"`

	ClassChain string `json:"class chain"`

	XPath string `json:"xpath"`
}

func (wl BySelector) getUsingAndValue() (using, value string) {
	vBy := reflect.ValueOf(wl)
	tBy := reflect.TypeOf(wl)
	for i := 0; i < vBy.NumField(); i++ {
		vi := vBy.Field(i).Interface()
		switch vi := vi.(type) {
		case ElementType:
			value = vi.String()
		case string:
			value = vi
		case ElementAttribute:
			value = vi.String()
		}
		if value != "" && value != "UNKNOWN" {
			using = tBy.Field(i).Tag.Get("json")
			return
		}
	}
	return
}

type ElementAttribute map[string]interface{}

func (ea ElementAttribute) String() string {
	for k, v := range ea {
		switch v := v.(type) {
		case bool:
			return k + "=" + strconv.FormatBool(v)
		case string:
			return k + "=" + v
		default:
			return k + "=" + fmt.Sprintf("%v", v)
		}
	}
	return "UNKNOWN"
}

func (ea ElementAttribute) getAttributeName() string {
	for k := range ea {
		return k
	}
	return "UNKNOWN"
}

func NewElementAttribute() ElementAttribute {
	return make(ElementAttribute)
}

// WithUID Element's unique identifier
func (ea ElementAttribute) WithUID(uid string) ElementAttribute {
	ea["UID"] = uid
	return ea
}

// WithAccessibilityContainer Whether element is an accessibility container
// (contains children of any depth that are accessible)
func (ea ElementAttribute) WithAccessibilityContainer(b bool) ElementAttribute {
	ea["accessibilityContainer"] = b
	return ea
}

// WithAccessible Whether element is accessible
func (ea ElementAttribute) WithAccessible(b bool) ElementAttribute {
	ea["accessible"] = b
	return ea
}

// WithEnabled Whether element is enabled
func (ea ElementAttribute) WithEnabled(b bool) ElementAttribute {
	ea["enabled"] = b
	return ea
}

// WithLabel Element's label
func (ea ElementAttribute) WithLabel(s string) ElementAttribute {
	ea["label"] = s
	return ea
}

// WithName Element's name
func (ea ElementAttribute) WithName(s string) ElementAttribute {
	ea["name"] = s
	return ea
}

// WithSelected Element's selected state
func (ea ElementAttribute) WithSelected(b bool) ElementAttribute {
	ea["selected"] = b
	return ea
}

// WithType Element's type
func (ea ElementAttribute) WithType(elemType ElementType) ElementAttribute {
	ea["type"] = elemType
	return ea
}

// WithValue Element's value
func (ea ElementAttribute) WithValue(s string) ElementAttribute {
	ea["value"] = s
	return ea
}

// WithVisible
//
// Whether element is visible
func (ea ElementAttribute) WithVisible(b bool) ElementAttribute {
	ea["visible"] = b
	return ea
}

func (et ElementType) String() string {
	vBy := reflect.ValueOf(et)
	tBy := reflect.TypeOf(et)
	for i := 0; i < vBy.NumField(); i++ {
		if vBy.Field(i).Bool() {
			return tBy.Field(i).Tag.Get("json")
		}
	}
	return "UNKNOWN"
}

// ElementType
// !!! This mapping should be updated if there are changes after each new XCTest release"`
type ElementType struct {
	Any                bool `json:"XCUIElementTypeAny"`
	Other              bool `json:"XCUIElementTypeOther"`
	Application        bool `json:"XCUIElementTypeApplication"`
	Group              bool `json:"XCUIElementTypeGroup"`
	Window             bool `json:"XCUIElementTypeWindow"`
	Sheet              bool `json:"XCUIElementTypeSheet"`
	Drawer             bool `json:"XCUIElementTypeDrawer"`
	Alert              bool `json:"XCUIElementTypeAlert"`
	Dialog             bool `json:"XCUIElementTypeDialog"`
	Button             bool `json:"XCUIElementTypeButton"`
	RadioButton        bool `json:"XCUIElementTypeRadioButton"`
	RadioGroup         bool `json:"XCUIElementTypeRadioGroup"`
	CheckBox           bool `json:"XCUIElementTypeCheckBox"`
	DisclosureTriangle bool `json:"XCUIElementTypeDisclosureTriangle"`
	PopUpButton        bool `json:"XCUIElementTypePopUpButton"`
	ComboBox           bool `json:"XCUIElementTypeComboBox"`
	MenuButton         bool `json:"XCUIElementTypeMenuButton"`
	ToolbarButton      bool `json:"XCUIElementTypeToolbarButton"`
	Popover            bool `json:"XCUIElementTypePopover"`
	Keyboard           bool `json:"XCUIElementTypeKeyboard"`
	Key                bool `json:"XCUIElementTypeKey"`
	NavigationBar      bool `json:"XCUIElementTypeNavigationBar"`
	TabBar             bool `json:"XCUIElementTypeTabBar"`
	TabGroup           bool `json:"XCUIElementTypeTabGroup"`
	Toolbar            bool `json:"XCUIElementTypeToolbar"`
	StatusBar          bool `json:"XCUIElementTypeStatusBar"`
	Table              bool `json:"XCUIElementTypeTable"`
	TableRow           bool `json:"XCUIElementTypeTableRow"`
	TableColumn        bool `json:"XCUIElementTypeTableColumn"`
	Outline            bool `json:"XCUIElementTypeOutline"`
	OutlineRow         bool `json:"XCUIElementTypeOutlineRow"`
	Browser            bool `json:"XCUIElementTypeBrowser"`
	CollectionView     bool `json:"XCUIElementTypeCollectionView"`
	Slider             bool `json:"XCUIElementTypeSlider"`
	PageIndicator      bool `json:"XCUIElementTypePageIndicator"`
	ProgressIndicator  bool `json:"XCUIElementTypeProgressIndicator"`
	ActivityIndicator  bool `json:"XCUIElementTypeActivityIndicator"`
	SegmentedControl   bool `json:"XCUIElementTypeSegmentedControl"`
	Picker             bool `json:"XCUIElementTypePicker"`
	PickerWheel        bool `json:"XCUIElementTypePickerWheel"`
	Switch             bool `json:"XCUIElementTypeSwitch"`
	Toggle             bool `json:"XCUIElementTypeToggle"`
	Link               bool `json:"XCUIElementTypeLink"`
	Image              bool `json:"XCUIElementTypeImage"`
	Icon               bool `json:"XCUIElementTypeIcon"`
	SearchField        bool `json:"XCUIElementTypeSearchField"`
	ScrollView         bool `json:"XCUIElementTypeScrollView"`
	ScrollBar          bool `json:"XCUIElementTypeScrollBar"`
	StaticText         bool `json:"XCUIElementTypeStaticText"`
	TextField          bool `json:"XCUIElementTypeTextField"`
	SecureTextField    bool `json:"XCUIElementTypeSecureTextField"`
	DatePicker         bool `json:"XCUIElementTypeDatePicker"`
	TextView           bool `json:"XCUIElementTypeTextView"`
	Menu               bool `json:"XCUIElementTypeMenu"`
	MenuItem           bool `json:"XCUIElementTypeMenuItem"`
	MenuBar            bool `json:"XCUIElementTypeMenuBar"`
	MenuBarItem        bool `json:"XCUIElementTypeMenuBarItem"`
	Map                bool `json:"XCUIElementTypeMap"`
	WebView            bool `json:"XCUIElementTypeWebView"`
	IncrementArrow     bool `json:"XCUIElementTypeIncrementArrow"`
	DecrementArrow     bool `json:"XCUIElementTypeDecrementArrow"`
	Timeline           bool `json:"XCUIElementTypeTimeline"`
	RatingIndicator    bool `json:"XCUIElementTypeRatingIndicator"`
	ValueIndicator     bool `json:"XCUIElementTypeValueIndicator"`
	SplitGroup         bool `json:"XCUIElementTypeSplitGroup"`
	Splitter           bool `json:"XCUIElementTypeSplitter"`
	RelevanceIndicator bool `json:"XCUIElementTypeRelevanceIndicator"`
	ColorWell          bool `json:"XCUIElementTypeColorWell"`
	HelpTag            bool `json:"XCUIElementTypeHelpTag"`
	Matte              bool `json:"XCUIElementTypeMatte"`
	DockItem           bool `json:"XCUIElementTypeDockItem"`
	Ruler              bool `json:"XCUIElementTypeRuler"`
	RulerMarker        bool `json:"XCUIElementTypeRulerMarker"`
	Grid               bool `json:"XCUIElementTypeGrid"`
	LevelIndicator     bool `json:"XCUIElementTypeLevelIndicator"`
	Cell               bool `json:"XCUIElementTypeCell"`
	LayoutArea         bool `json:"XCUIElementTypeLayoutArea"`
	LayoutItem         bool `json:"XCUIElementTypeLayoutItem"`
	Handle             bool `json:"XCUIElementTypeHandle"`
	Stepper            bool `json:"XCUIElementTypeStepper"`
	Tab                bool `json:"XCUIElementTypeTab"`
	TouchBar           bool `json:"XCUIElementTypeTouchBar"`
	StatusItem         bool `json:"XCUIElementTypeStatusItem"`
}

// ProtectedResource A system resource that requires user authorization to access.
type ProtectedResource int

// https://developer.apple.com/documentation/xctest/xcuiprotectedresource?language=objc
const (
	ProtectedResourceContacts               ProtectedResource = 1
	ProtectedResourceCalendar               ProtectedResource = 2
	ProtectedResourceReminders              ProtectedResource = 3
	ProtectedResourcePhotos                 ProtectedResource = 4
	ProtectedResourceMicrophone             ProtectedResource = 5
	ProtectedResourceCamera                 ProtectedResource = 6
	ProtectedResourceMediaLibrary           ProtectedResource = 7
	ProtectedResourceHomeKit                ProtectedResource = 8
	ProtectedResourceSystemRootDirectory    ProtectedResource = 0x40000000
	ProtectedResourceUserDesktopDirectory   ProtectedResource = 0x40000001
	ProtectedResourceUserDownloadsDirectory ProtectedResource = 0x40000002
	ProtectedResourceUserDocumentsDirectory ProtectedResource = 0x40000003
	ProtectedResourceBluetooth              ProtectedResource = -0x40000000
	ProtectedResourceKeyboardNetwork        ProtectedResource = -0x40000001
	ProtectedResourceLocation               ProtectedResource = -0x40000002
	ProtectedResourceHealth                 ProtectedResource = -0x40000003
)

type Condition func(wd WebDriver) (bool, error)

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type PickerWheelOrder string

const (
	PickerWheelOrderNext     PickerWheelOrder = "next"
	PickerWheelOrderPrevious PickerWheelOrder = "previous"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Rect struct {
	Point
	Size
}

// WebDriver defines methods supported by WebDriver drivers.
type WebDriver interface {
	// NewSession starts a new session and returns the SessionInfo.
	NewSession(capabilities Capabilities) (SessionInfo, error)

	ActiveSession() (SessionInfo, error)
	// DeleteSession Kills application associated with that session and removes session
	//  1) alertsMonitor disable
	//  2) testedApplicationBundleId terminate
	DeleteSession() error

	Status() (DeviceStatus, error)

	DeviceInfo() (DeviceInfo, error)

	// Location Returns device location data.
	//
	// It requires to configure location access permission by manual.
	// The response of 'latitude', 'longitude' and 'altitude' are always zero (0) without authorization.
	// 'authorizationStatus' indicates current authorization status. '3' is 'Always'.
	// https://developer.apple.com/documentation/corelocation/clauthorizationstatus
	//
	//  Settings -> Privacy -> Location Service -> WebDriverAgent-Runner -> Always
	//
	// The return value could be zero even if the permission is set to 'Always'
	// since the location service needs some time to update the location data.
	Location() (Location, error)
	BatteryInfo() (BatteryInfo, error)
	WindowSize() (Size, error)
	Screen() (Screen, error)
	Scale() (float64, error)
	ActiveAppInfo() (AppInfo, error)
	// ActiveAppsList Retrieves the information about the currently active apps
	ActiveAppsList() ([]AppBaseInfo, error)
	// AppState Get the state of the particular application in scope of the current session.
	// !This method is only returning reliable results since Xcode9 SDK
	AppState(bundleId string) (AppState, error)

	// IsLocked Checks if the screen is locked or not.
	IsLocked() (bool, error)
	// Unlock Forces the device under test to unlock.
	// An immediate return will happen if the device is already unlocked
	// and an error is going to be thrown if the screen has not been unlocked after the timeout.
	Unlock() error
	// Lock Forces the device under test to switch to the lock screen.
	// An immediate return will happen if the device is already locked
	// and an error is going to be thrown if the screen has not been locked after the timeout.
	Lock() error

	// Homescreen Forces the device under test to switch to the home screen
	Homescreen() error

	// AlertText Returns alert's title and description separated by new lines
	AlertText() (string, error)
	// AlertButtons Gets the labels of the buttons visible in the alert
	AlertButtons() ([]string, error)
	// AlertAccept Accepts alert, if present
	AlertAccept(label ...string) error
	// AlertDismiss Dismisses alert, if present
	AlertDismiss(label ...string) error
	// AlertSendKeys Types a text into an input inside the alert container, if it is present
	AlertSendKeys(text string) error

	// AppLaunch Launch an application with given bundle identifier in scope of current session.
	// !This method is only available since Xcode9 SDK
	AppLaunch(bundleId string, launchOpt ...AppLaunchOption) error
	// AppLaunchUnattached Launch the app with the specified bundle ID.
	AppLaunchUnattached(bundleId string) error
	// AppTerminate Terminate an application with the given bundle id.
	// Either `true` if the app has been successfully terminated or `false` if it was not running
	AppTerminate(bundleId string) (bool, error)
	// AppActivate Activate an application with given bundle identifier in scope of current session.
	// !This method is only available since Xcode9 SDK
	AppActivate(bundleId string) error
	// AppDeactivate Deactivates application for given time and then activate it again
	//  The minimum application switch wait is 3 seconds
	AppDeactivate(second float64) error

	// AppAuthReset Resets the authorization status for a protected resource. Available since Xcode 11.4
	AppAuthReset(ProtectedResource) error

	// Tap Sends a tap event at the coordinate.
	Tap(x, y int) error
	TapFloat(x, y float64) error

	// DoubleTap Sends a double tap event at the coordinate.
	DoubleTap(x, y int) error
	DoubleTapFloat(x, y float64) error

	// TouchAndHold Initiates a long-press gesture at the coordinate, holding for the specified duration.
	//  second: The default value is 1
	TouchAndHold(x, y int, second ...float64) error
	TouchAndHoldFloat(x, y float64, second ...float64) error

	// Drag Initiates a press-and-hold gesture at the coordinate, then drags to another coordinate.
	//  pressForDuration: The default value is 1 second.
	Drag(fromX, fromY, toX, toY int, pressForDuration ...float64) error
	DragFloat(fromX, fromY, toX, toY float64, pressForDuration ...float64) error

	// Swipe works like Drag, but `pressForDuration` value is 0
	Swipe(fromX, fromY, toX, toY int) error
	SwipeFloat(fromX, fromY, toX, toY float64) error

	ForceTouch(x, y int, pressure float64, second ...float64) error
	ForceTouchFloat(x, y, pressure float64, second ...float64) error

	// PerformW3CActions Perform complex touch action in scope of the current application.
	PerformW3CActions(actions *W3CActions) error
	PerformAppiumTouchActions(touchActs *TouchActions) error

	// SetPasteboard Sets data to the general pasteboard
	SetPasteboard(contentType PasteboardType, content string) error
	// GetPasteboard Gets the data contained in the general pasteboard.
	//  It worked when `WDA` was foreground. https://github.com/appium/WebDriverAgent/issues/330
	GetPasteboard(contentType PasteboardType) (raw *bytes.Buffer, err error)

	// SendKeys Types a string into active element. There must be element with keyboard focus,
	// otherwise an error is raised.
	//  frequency: Frequency of typing (letters per sec). The default value is 60
	SendKeys(text string, frequency ...int) error

	// KeyboardDismiss Tries to dismiss the on-screen keyboard
	KeyboardDismiss(keyNames ...string) error

	// PressButton Presses the corresponding hardware button on the device
	PressButton(devBtn DeviceButton) error

	// IOHIDEvent Emulated triggering of the given low-level IOHID device event.
	//  duration: The event duration in float seconds (XCTest uses 0.005 for a single press event)
	IOHIDEvent(pageID EventPageID, usageID EventUsageID, duration ...float64) error

	// ExpectNotification Creates an expectation that is fulfilled when an expected Notification is received
	ExpectNotification(notifyName string, notifyType NotificationType, second ...int) error

	// SiriActivate Activates Siri service voice recognition with the given text to parse
	SiriActivate(text string) error
	// SiriOpenUrl Opens the particular url scheme using Siri voice recognition helpers.
	// !This will only work since XCode 8.3/iOS 10.3
	//  It doesn't actually work, right?
	SiriOpenUrl(url string) error

	Orientation() (Orientation, error)
	// SetOrientation Sets requested device interface orientation.
	SetOrientation(Orientation) error

	Rotation() (Rotation, error)
	// SetRotation Sets the devices orientation to the rotation passed.
	SetRotation(Rotation) error

	// MatchTouchID Matches or mismatches TouchID request
	MatchTouchID(isMatch bool) error

	// ActiveElement Returns the element, which currently holds the keyboard input focus or nil if there are no such elements.
	ActiveElement() (WebElement, error)
	FindElement(by BySelector) (WebElement, error)
	FindElements(by BySelector) ([]WebElement, error)

	Screenshot() (*bytes.Buffer, error)

	// Source Return application elements tree
	Source(srcOpt ...SourceOption) (string, error)
	// AccessibleSource Return application elements accessibility tree
	AccessibleSource() (string, error)

	// HealthCheck Health check might modify simulator state so it should only be called in-between testing sessions
	//  Checks health of XCTest by:
	//  1) Querying application for some elements,
	//  2) Triggering some device events.
	HealthCheck() error
	GetAppiumSettings() (map[string]interface{}, error)
	SetAppiumSettings(settings map[string]interface{}) (map[string]interface{}, error)

	IsWdaHealthy() (bool, error)
	WdaShutdown() error

	// WaitWithTimeoutAndInterval waits for the condition to evaluate to true.
	WaitWithTimeoutAndInterval(condition Condition, timeout, interval time.Duration) error
	// WaitWithTimeout works like WaitWithTimeoutAndInterval, but with default polling interval.
	WaitWithTimeout(condition Condition, timeout time.Duration) error
	// Wait works like WaitWithTimeoutAndInterval, but using the default timeout and polling interval.
	Wait(condition Condition) error

	GetMjpegHTTPClient() *http.Client

	//uusense
	Dragfromtoforduration(fromX, fromY, toX, toY float64, duration float64) (err error)
	DoubleMove(aX1, aY1, aX2, aY2, bX1, bY1, bX2, bY2 float64, duration float64) (err error)
	SlidePath(points []map[string]int, duration float64) (err error)
	ScreenshotUUSense(shotType int, X float64, Y float64, width float64, height float64, quality int) (raw *bytes.Buffer, err error)
	InputUUSense(test string) (err error)
	GetDeviceInfo() (ret StatusInfo, err error)
}

// WebElement defines method supported by web elements.
type WebElement interface {
	// Click Waits for element to become stable (not move) and performs sync tap on element.
	Click() error
	// SendKeys Types a text into element. It will try to activate keyboard on element,
	// if element has no keyboard focus.
	//  frequency: Frequency of typing (letters per sec). The default value is 60
	SendKeys(text string, frequency ...int) error
	// Clear Clears text on element. It will try to activate keyboard on element,
	// if element has no keyboard focus.
	Clear() error

	// Tap Waits for element to become stable (not move) and performs sync tap on element,
	// relative to the current element position
	Tap(x, y int) error
	TapFloat(x, y float64) error

	// DoubleTap Sends a double tap event to a hittable point computed for the element.
	DoubleTap() error

	// TouchAndHold Sends a long-press gesture to a hittable point computed for the element,
	// holding for the specified duration.
	//  second: The default value is 1
	TouchAndHold(second ...float64) error
	// TwoFingerTap Sends a two finger tap event to a hittable point computed for the element.
	TwoFingerTap() error
	// TapWithNumberOfTaps Sends one or more taps with one or more touch points.
	TapWithNumberOfTaps(numberOfTaps, numberOfTouches int) error
	// ForceTouch Waits for element to become stable (not move) and performs sync force touch on element.
	//  second: The default value is 1
	ForceTouch(pressure float64, second ...float64) error
	// ForceTouchFloat works like ForceTouch, but relative to the current element position
	ForceTouchFloat(x, y, pressure float64, second ...float64) error

	// Drag Initiates a press-and-hold gesture at the coordinate, then drags to another coordinate.
	// relative to the current element position
	//  pressForDuration: The default value is 1 second.
	Drag(fromX, fromY, toX, toY int, pressForDuration ...float64) error
	DragFloat(fromX, fromY, toX, toY float64, pressForDuration ...float64) error

	// Swipe works like Drag, but `pressForDuration` value is 0.
	// relative to the current element position
	Swipe(fromX, fromY, toX, toY int) error
	SwipeFloat(fromX, fromY, toX, toY float64) error
	// SwipeDirection Performs swipe gesture on the element.
	//  velocity: swipe speed in pixels per second. Custom velocity values are only supported since Xcode SDK 11.4.
	SwipeDirection(direction Direction, velocity ...float64) error

	// Pinch Sends a pinching gesture with two touches.
	//  scale: The scale of the pinch gesture. Use a scale between 0 and 1 to "pinch close" or zoom out
	//  and a scale greater than 1 to "pinch open" or zoom in.
	//  velocity: The velocity of the pinch in scale factor per second.
	Pinch(scale, velocity float64) error
	PinchToZoomOutByW3CAction(scale ...float64) error

	// Rotate Sends a rotation gesture with two touches.
	//  rotation: The rotation of the gesture in radians.
	//  velocity: The velocity of the rotation gesture in radians per second.
	Rotate(rotation float64, velocity ...float64) error

	// PickerWheelSelect
	//  offset: The default value is 2
	PickerWheelSelect(order PickerWheelOrder, offset ...int) error

	ScrollElementByName(name string) error
	ScrollElementByPredicate(predicate string) error
	ScrollToVisible() error
	// ScrollDirection
	//  distance: The default value is 0.5
	ScrollDirection(direction Direction, distance ...float64) error

	FindElement(by BySelector) (element WebElement, err error)
	FindElements(by BySelector) (elements []WebElement, err error)
	FindVisibleCells() (elements []WebElement, err error)

	Rect() (rect Rect, err error)
	Location() (Point, error)
	Size() (Size, error)
	Text() (text string, err error)
	Type() (elemType string, err error)
	IsEnabled() (enabled bool, err error)
	IsDisplayed() (displayed bool, err error)
	IsSelected() (selected bool, err error)
	IsAccessible() (accessible bool, err error)
	IsAccessibilityContainer() (isAccessibilityContainer bool, err error)
	GetAttribute(attr ElementAttribute) (value string, err error)
	UID() (uid string)

	Screenshot() (raw *bytes.Buffer, err error)
}
