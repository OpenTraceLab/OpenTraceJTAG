package ui

import (
	"sync"
	"time"

	"github.com/OpenTraceLab/OpenTraceJTAG/pkg/jtag"
)

// JTAGDevice describes a device discovered on the boundary scan chain.
type JTAGDevice struct {
	Index    int
	IDCode   uint32
	Name     string
	Package  string
	BSDLFile string
}

// StateSnapshot captures a copy of the state data for rendering without
// requiring the UI to hold locks while laying out widgets.
type StateSnapshot struct {
	AdapterInfo *jtag.AdapterInfo
	Connected   bool
	Busy        bool

	LastError error
	Status    string

	Chain       []JTAGDevice
	SelectedIdx int

	Interfaces        []jtag.InterfaceInfo
	SelectedInterface int
	DriverKind        jtag.InterfaceKind

	SelectedView      appView
	LeftPanelVisible  bool
	RightPanelVisible bool
	AppVersion        string

	Logs []string

	LastUpdated time.Time
}

// AppState tracks the mutable state shared between the Gio event loop and
// background goroutines performing boundary scan operations.
type AppState struct {
	mu sync.RWMutex

	adapter     jtag.Adapter
	adapterInfo *jtag.AdapterInfo
	connected   bool
	busy        bool

	lastError error
	status    string

	chain       []JTAGDevice
	selectedIdx int

	interfaces        []jtag.InterfaceInfo
	selectedInterface int
	driverKind        jtag.InterfaceKind

	selectedView      appView
	leftPanelVisible  bool
	rightPanelVisible bool
	appVersion        string

	logs     []string
	logLimit int

	lastUpdated time.Time
}

// NewState returns a baseline AppState with safe defaults.
func NewState() *AppState {
	return &AppState{
		selectedIdx:       -1,
		selectedInterface: -1,
		driverKind:        jtag.InterfaceKindUnknown,
		logLimit:          200,
		status:            "Idle",
		selectedView:      viewDrivers,
		leftPanelVisible:  true,
		rightPanelVisible: true,
		appVersion:        "dev",
		lastUpdated:       time.Now(),
	}
}

// Snapshot returns a copy of the mutable state for rendering.
func (s *AppState) Snapshot() StateSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var infoCopy *jtag.AdapterInfo
	if s.adapterInfo != nil {
		info := *s.adapterInfo
		infoCopy = &info
	}

	chainCopy := make([]JTAGDevice, len(s.chain))
	copy(chainCopy, s.chain)

	logCopy := make([]string, len(s.logs))
	copy(logCopy, s.logs)

	return StateSnapshot{
		AdapterInfo: infoCopy,
		Connected:   s.connected,
		Busy:        s.busy,
		LastError:   s.lastError,
		Status:      s.status,
		Chain:       chainCopy,
		SelectedIdx: s.selectedIdx,
		DriverKind:  s.driverKind,
		Interfaces: func() []jtag.InterfaceInfo {
			if len(s.interfaces) == 0 {
				return nil
			}
			clone := make([]jtag.InterfaceInfo, len(s.interfaces))
			copy(clone, s.interfaces)
			return clone
		}(),
		SelectedInterface: s.selectedInterface,
		Logs:              logCopy,
		SelectedView:      s.selectedView,
		LeftPanelVisible:  s.leftPanelVisible,
		RightPanelVisible: s.rightPanelVisible,
		AppVersion:        s.appVersion,
		LastUpdated:       s.lastUpdated,
	}
}

// SetAdapter attaches a new adapter reference and optional info.
func (s *AppState) SetAdapter(adapter jtag.Adapter, info *jtag.AdapterInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.adapter = adapter
	if info != nil {
		clone := *info
		s.adapterInfo = &clone
	} else {
		s.adapterInfo = nil
	}
	s.connected = adapter != nil
	s.lastUpdated = time.Now()
}

// Adapter returns the currently attached adapter, if any.
func (s *AppState) Adapter() jtag.Adapter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.adapter
}

// SetChain replaces the chain device list and adjusts the selection cursor.
func (s *AppState) SetChain(devices []JTAGDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chain = make([]JTAGDevice, len(devices))
	copy(s.chain, devices)

	if len(s.chain) == 0 {
		s.selectedIdx = -1
	} else if s.selectedIdx < 0 || s.selectedIdx >= len(s.chain) {
		s.selectedIdx = 0
	}
	s.lastUpdated = time.Now()
}

// SelectDevice moves the selection cursor to the provided index if valid.
func (s *AppState) SelectDevice(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx < 0 || idx >= len(s.chain) {
		return
	}
	s.selectedIdx = idx
	s.lastUpdated = time.Now()
}

// SelectedDevice returns the currently selected chain entry, if any.
func (s *AppState) SelectedDevice() *JTAGDevice {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.selectedIdx < 0 || s.selectedIdx >= len(s.chain) {
		return nil
	}
	dev := s.chain[s.selectedIdx]
	return &dev
}

// SetBusy toggles the busy flag and updates the timestamp.
func (s *AppState) SetBusy(busy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.busy = busy
	s.lastUpdated = time.Now()
}

// Busy returns the current busy flag.
func (s *AppState) Busy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.busy
}

// SetStatus updates the user-facing status message.
func (s *AppState) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.lastUpdated = time.Now()
}

// SetError stores the latest error surfaced to the UI.
func (s *AppState) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
	s.lastUpdated = time.Now()
}

// AppendLog appends a log message, trimming the oldest entries past the limit.
func (s *AppState) AppendLog(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, msg)
	if s.logLimit > 0 && len(s.logs) > s.logLimit {
		offset := len(s.logs) - s.logLimit
		s.logs = append([]string(nil), s.logs[offset:]...)
	}
	s.lastUpdated = time.Now()
}

// SetInterfaces records the list of detected interfaces.
func (s *AppState) SetInterfaces(infos []jtag.InterfaceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interfaces = make([]jtag.InterfaceInfo, len(infos))
	copy(s.interfaces, infos)
	if len(s.interfaces) == 0 {
		s.selectedInterface = -1
	} else if s.selectedInterface < 0 || s.selectedInterface >= len(s.interfaces) {
		s.selectedInterface = 0
	}
	s.lastUpdated = time.Now()
}

// Interfaces returns the currently recorded interfaces.
func (s *AppState) Interfaces() []jtag.InterfaceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := make([]jtag.InterfaceInfo, len(s.interfaces))
	copy(clone, s.interfaces)
	return clone
}

// SelectInterface sets the active interface index if valid.
func (s *AppState) SelectInterface(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx < 0 || idx >= len(s.interfaces) {
		return
	}
	s.selectedInterface = idx
	s.lastUpdated = time.Now()
}

// SelectedInterface returns the currently selected interface, if any.
func (s *AppState) SelectedInterface() *jtag.InterfaceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.selectedInterface < 0 || s.selectedInterface >= len(s.interfaces) {
		return nil
	}
	info := s.interfaces[s.selectedInterface]
	return &info
}

// SetAppVersion records the running UI/application version string.
func (s *AppState) SetAppVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version == "" {
		version = "dev"
	}
	s.appVersion = version
	s.lastUpdated = time.Now()
}

// AppVersion returns the current application version string.
func (s *AppState) AppVersion() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.appVersion
}

// SetDriverKind records the adapter driver kind currently chosen.
func (s *AppState) SetDriverKind(kind jtag.InterfaceKind) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.driverKind == kind {
		return
	}
	s.driverKind = kind
	s.lastUpdated = time.Now()
}

// DriverKind reports the active adapter driver kind filter.
func (s *AppState) DriverKind() jtag.InterfaceKind {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.driverKind
}

// SetView updates the active workspace selection.
func (s *AppState) SetView(view appView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selectedView == view {
		return
	}
	s.selectedView = view
	s.lastUpdated = time.Now()
}

// SelectedView reports the current workspace selection.
func (s *AppState) SelectedView() appView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selectedView
}

// SetLeftPanelVisible toggles the left panel visibility bit.
func (s *AppState) SetLeftPanelVisible(visible bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leftPanelVisible == visible {
		return
	}
	s.leftPanelVisible = visible
	s.lastUpdated = time.Now()
}

// LeftPanelVisible returns whether the left dock should render.
func (s *AppState) LeftPanelVisible() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leftPanelVisible
}

// SetRightPanelVisible toggles the right panel visibility bit.
func (s *AppState) SetRightPanelVisible(visible bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rightPanelVisible == visible {
		return
	}
	s.rightPanelVisible = visible
	s.lastUpdated = time.Now()
}

// RightPanelVisible returns whether the right dock should render.
func (s *AppState) RightPanelVisible() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rightPanelVisible
}
