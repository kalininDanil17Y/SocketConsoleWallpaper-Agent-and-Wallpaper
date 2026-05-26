//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/kardianos/service"
	"github.com/lxn/win"

	"socket-console-agent/internal/api"
	"socket-console-agent/internal/config"
	"socket-console-agent/internal/metrics"
)

const (
	mainClassName   = "SocketConsoleAgentControlPanel"
	configClassName = "SocketConsoleAgentConfigDialog"

	idConfigPath  = 100
	idRun         = 101
	idStart       = 102
	idStop        = 103
	idInstall     = 104
	idUninstall   = 105
	idStatus      = 106
	idConfig      = 107
	idOpenFolder  = 108
	idReinstall   = 109
	idLogs        = 110
	idStatusText  = 111
	idApplyConfig = 112

	idCfgPort         = 200
	idCfgInterval     = 201
	idCfgCPU          = 202
	idCfgMemory       = 203
	idCfgDisks        = 204
	idCfgNetwork      = 205
	idCfgGPU          = 206
	idCfgTemperatures = 207
	idCfgScreens      = 208
	idCfgInterface    = 209
	idCfgPreferIPv4   = 210
	idCfgImageDir     = 212
	idCfgChangeEvery  = 213
	idCfgASCIIWidth   = 214
	idCfgASCIIHeight  = 215
	idCfgCharset      = 216
	idCfgPalette      = 217
	idCfgSave         = 218
	idCfgCancel       = 219

	wmAsyncRefresh   = win.WM_APP + 1
	wmAsyncState     = win.WM_APP + 2
	wmRequestRefresh = win.WM_APP + 3
)

type controlPanel struct {
	mu      sync.Mutex
	service service.Service
	server  *api.Server
	cancel  context.CancelFunc
	running bool

	hwnd            win.HWND
	configDlg       *configDialog
	statusText      win.HWND
	configPathEdit  win.HWND
	logsEdit        win.HWND
	runButton       win.HWND
	startButton     win.HWND
	stopButton      win.HWND
	installButton   win.HWND
	uninstallButton win.HWND
	reinstallButton win.HWND
	pendingLogs     []string
	status          service.Status
	installed       bool
	statusErr       error
	statusKnown     bool
	refreshing      bool
}

type configDialog struct {
	panel controlPanelRef
	hwnd  win.HWND

	portEdit        win.HWND
	intervalEdit    win.HWND
	cpuCB           win.HWND
	memoryCB        win.HWND
	disksCB         win.HWND
	networkCB       win.HWND
	gpuCB           win.HWND
	temperaturesCB  win.HWND
	screensCB       win.HWND
	interfaceCombo  win.HWND
	preferIPv4CB    win.HWND
	diskChecks      []diskCheck
	imageDirEdit    win.HWND
	changeEveryEdit win.HWND
	asciiWidthEdit  win.HWND
	asciiHeightEdit win.HWND
	charsetEdit     win.HWND
	paletteEdit     win.HWND
}

type diskCheck struct {
	name string
	hwnd win.HWND
}

type controlPanelRef struct {
	*controlPanel
}

var (
	hInstance     win.HINSTANCE
	currentPanel  *controlPanel
	currentConfig *configDialog
)

func runUI(svc service.Service) error {
	runtime.LockOSThread()

	hInstance = win.GetModuleHandle(nil)
	if err := registerWindowClasses(); err != nil {
		return err
	}

	panel := &controlPanel{service: svc}
	currentPanel = panel

	hwnd := win.CreateWindowEx(
		0,
		utf16Ptr(mainClassName),
		utf16Ptr("Socket Console Agent"),
		win.WS_OVERLAPPEDWINDOW,
		win.CW_USEDEFAULT,
		win.CW_USEDEFAULT,
		780,
		560,
		0,
		0,
		hInstance,
		nil,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx failed")
	}
	panel.hwnd = hwnd

	win.ShowWindow(hwnd, win.SW_SHOW)
	win.UpdateWindow(hwnd)
	panel.appendLog("UI started.")
	panel.refreshAsync()

	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
	return nil
}

func registerWindowClasses() error {
	cursor := win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW))
	for _, className := range []string{mainClassName, configClassName} {
		wc := win.WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
			LpfnWndProc:   syscall.NewCallback(windowProc),
			HInstance:     hInstance,
			HCursor:       cursor,
			HbrBackground: win.HBRUSH(win.COLOR_WINDOW + 1),
			LpszClassName: utf16Ptr(className),
		}
		if atom := win.RegisterClassEx(&wc); atom == 0 {
			return fmt.Errorf("RegisterClassEx(%s) failed", className)
		}
	}
	return nil
}

func windowProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_CREATE:
		if currentPanel != nil && currentPanel.hwnd == 0 {
			currentPanel.hwnd = hwnd
			currentPanel.createMainControls(hwnd)
			return 0
		}
		if currentConfig != nil && currentConfig.hwnd == 0 {
			currentConfig.hwnd = hwnd
			currentConfig.createControls(hwnd)
			return 0
		}
	case win.WM_COMMAND:
		id := int(win.LOWORD(uint32(wParam)))
		if currentConfig != nil && hwnd == currentConfig.hwnd {
			currentConfig.handleCommand(id)
			return 0
		}
		if currentPanel != nil && hwnd == currentPanel.hwnd {
			currentPanel.handleCommand(id)
			return 0
		}
	case wmAsyncRefresh:
		if currentPanel != nil {
			currentPanel.flushPendingLogs()
		}
		return 0
	case wmAsyncState:
		if currentPanel != nil {
			currentPanel.applyState()
		}
		return 0
	case wmRequestRefresh:
		if currentPanel != nil {
			currentPanel.refreshAsync()
		}
		return 0
	case win.WM_CLOSE:
		if currentConfig != nil && hwnd == currentConfig.hwnd {
			currentConfig.close()
			return 0
		}
		if currentPanel != nil && hwnd == currentPanel.hwnd {
			currentPanel.stopLocalRun()
			win.DestroyWindow(hwnd)
			return 0
		}
	case win.WM_DESTROY:
		if currentPanel != nil && hwnd == currentPanel.hwnd {
			win.PostQuitMessage(0)
			return 0
		}
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (p *controlPanel) createMainControls(hwnd win.HWND) {
	p.label(hwnd, "Service status:", 16, 18, 110, 22)
	p.statusText = p.label(hwnd, "Checking...", 130, 18, 600, 22)
	p.label(hwnd, "Service config path:", 16, 50, 130, 22)
	p.configPathEdit = p.edit(hwnd, config.ServiceConfigPath(), 150, 48, 590, 24, idConfigPath)

	p.runButton = p.button(hwnd, "Run locally", 16, 88, 110, 30, idRun)
	p.startButton = p.button(hwnd, "Start service", 134, 88, 110, 30, idStart)
	p.stopButton = p.button(hwnd, "Stop service", 252, 88, 110, 30, idStop)
	p.installButton = p.button(hwnd, "Install", 370, 88, 90, 30, idInstall)
	p.uninstallButton = p.button(hwnd, "Uninstall", 468, 88, 100, 30, idUninstall)

	p.button(hwnd, "Status", 16, 128, 90, 30, idStatus)
	p.button(hwnd, "Config...", 114, 128, 90, 30, idConfig)
	p.button(hwnd, "Open config folder", 212, 128, 140, 30, idOpenFolder)
	p.reinstallButton = p.button(hwnd, "Reinstall with config path", 360, 128, 190, 30, idReinstall)
	p.button(hwnd, "Apply config", 558, 128, 120, 30, idApplyConfig)

	p.label(hwnd, "Logs:", 16, 174, 80, 22)
	p.logsEdit = p.multilineEdit(hwnd, "", 16, 198, 724, 250, idLogs, true)
	p.label(hwnd, "Important: install registers the service with the current exe path and selected config path. Keep the exe in a permanent folder.", 16, 470, 724, 38)
}

func (p *controlPanel) handleCommand(id int) {
	switch id {
	case idRun:
		p.toggleLocalRun()
	case idStart:
		p.startServiceAsync()
	case idStop:
		p.stopServiceAsync()
	case idInstall:
		p.installServiceAsync()
	case idUninstall:
		p.uninstallServiceAsync()
	case idStatus:
		p.refreshAsync()
	case idConfig:
		p.openConfigDialog()
	case idOpenFolder:
		p.openConfigFolder()
	case idReinstall:
		p.reinstallServiceAsync()
	case idApplyConfig:
		p.applyConfigAsync()
	}
}

func (p *controlPanel) configPath() string {
	path := strings.TrimSpace(windowText(p.configPathEdit))
	if path == "" {
		return config.ServiceConfigPath()
	}
	return path
}

func (p *controlPanel) appendLog(format string, args ...interface{}) {
	if p.logsEdit == 0 {
		return
	}
	line := time.Now().Format("15:04:05") + "  " + fmt.Sprintf(format, args...) + "\r\n"
	appendEditText(p.logsEdit, line)
}

func (p *controlPanel) appendLogAsync(format string, args ...interface{}) {
	p.mu.Lock()
	p.pendingLogs = append(p.pendingLogs, fmt.Sprintf(format, args...))
	p.mu.Unlock()
	win.PostMessage(p.hwnd, wmAsyncRefresh, 0, 0)
}

func (p *controlPanel) flushPendingLogs() {
	p.mu.Lock()
	logs := append([]string(nil), p.pendingLogs...)
	p.pendingLogs = nil
	p.mu.Unlock()
	for _, line := range logs {
		p.appendLog("%s", line)
	}
}

func (p *controlPanel) refreshAsync() {
	p.mu.Lock()
	if p.refreshing {
		p.mu.Unlock()
		return
	}
	p.refreshing = true
	p.mu.Unlock()

	setWindowText(p.statusText, "checking...")
	p.applyButtons()
	go func() {
		status, installed, err := serviceState(p.service)
		p.mu.Lock()
		p.status = status
		p.installed = installed
		p.statusErr = err
		p.statusKnown = true
		p.refreshing = false
		p.mu.Unlock()
		win.PostMessage(p.hwnd, wmAsyncState, 0, 0)
	}()
}

func (p *controlPanel) applyState() {
	p.mu.Lock()
	known := p.statusKnown
	installed := p.installed
	status := p.status
	err := p.statusErr
	p.mu.Unlock()

	if !known {
		setWindowText(p.statusText, "unknown")
	} else if installed {
		setWindowText(p.statusText, serviceStatusText(status))
	} else {
		setWindowText(p.statusText, "not installed")
		if err != nil {
			p.appendLog("Service is not installed or status is unavailable: %v", err)
		}
	}
	p.applyButtons()
}

func (p *controlPanel) applyButtons() {
	p.mu.Lock()
	localRunning := p.running
	installed := p.installed
	status := p.status
	refreshing := p.refreshing
	p.mu.Unlock()

	enableWindow(p.runButton, !installed && !refreshing)
	if localRunning {
		setWindowText(p.runButton, "Stop local run")
	} else {
		setWindowText(p.runButton, "Run locally")
	}
	enableWindow(p.startButton, installed && status != service.StatusRunning && !refreshing)
	enableWindow(p.stopButton, installed && status == service.StatusRunning && !refreshing)
	enableWindow(p.installButton, !installed && !refreshing)
	enableWindow(p.uninstallButton, installed && !refreshing)
	enableWindow(p.reinstallButton, installed && !refreshing)
}

func (p *controlPanel) toggleLocalRun() {
	p.mu.Lock()
	running := p.running
	p.mu.Unlock()
	if running {
		p.stopLocalRun()
		p.refreshAsync()
		return
	}
	p.mu.Lock()
	installed := p.installed
	p.mu.Unlock()
	if installed {
		p.appendLog("Local run is disabled while the service is installed. Use Start service instead.")
		p.refreshAsync()
		return
	}

	cfg, cfgPath, err := config.Load(p.configPath())
	if err != nil {
		p.showError("Run locally", err)
		return
	}
	srv := api.NewServer(cfg, cfgPath)
	ctx, cancel := context.WithCancel(context.Background())

	p.mu.Lock()
	p.server = srv
	p.cancel = cancel
	p.running = true
	p.mu.Unlock()

	p.appendLog("Local agent starting on http://%s:%d with config %s", cfg.Server.Host, cfg.Server.Port, cfgPath)
	go func() {
		err := srv.Run(ctx)
		p.mu.Lock()
		p.server = nil
		p.cancel = nil
		p.running = false
		p.mu.Unlock()
		if err != nil {
			p.appendLogAsync("Local agent stopped with error: %v", err)
			win.PostMessage(p.hwnd, wmRequestRefresh, 0, 0)
			return
		}
		p.appendLogAsync("Local agent stopped.")
		win.PostMessage(p.hwnd, wmRequestRefresh, 0, 0)
	}()
	p.applyButtons()
}

func (p *controlPanel) stopLocalRun() {
	p.mu.Lock()
	cancel := p.cancel
	server := p.server
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if server != nil {
		ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ctxCancel()
		_ = server.Shutdown(ctx)
	}
}

func (p *controlPanel) startServiceAsync() {
	p.serviceOp("Start service", func() error { return p.service.Start() })
}

func (p *controlPanel) stopServiceAsync() {
	p.serviceOp("Stop service", func() error { return p.service.Stop() })
}

func (p *controlPanel) installServiceAsync() {
	cfgPath := p.configPath()
	p.serviceOp("Install service", func() error {
		if err := ensureConfigExists(cfgPath); err != nil {
			return err
		}
		svc, err := newAgentService(serviceArgumentsForConfigPath(cfgPath))
		if err != nil {
			return err
		}
		return svc.Install()
	})
}

func (p *controlPanel) uninstallServiceAsync() {
	p.serviceOp("Uninstall service", func() error {
		status, installed, _ := serviceState(p.service)
		if installed && status == service.StatusRunning {
			_ = p.service.Stop()
		}
		return p.service.Uninstall()
	})
}

func (p *controlPanel) reinstallServiceAsync() {
	cfgPath := p.configPath()
	p.serviceOp("Reinstall service", func() error {
		status, installed, _ := serviceState(p.service)
		if installed && status == service.StatusRunning {
			if err := p.service.Stop(); err != nil {
				return err
			}
		}
		if installed {
			if err := p.service.Uninstall(); err != nil {
				return err
			}
		}
		if err := ensureConfigExists(cfgPath); err != nil {
			return err
		}
		svc, err := newAgentService(serviceArgumentsForConfigPath(cfgPath))
		if err != nil {
			return err
		}
		return svc.Install()
	})
}

func (p *controlPanel) applyConfigAsync() {
	p.mu.Lock()
	localRunning := p.running
	installed := p.installed
	status := p.status
	p.mu.Unlock()

	if localRunning {
		p.appendLog("Applying config by restarting local run.")
		p.stopLocalRun()
		time.AfterFunc(300*time.Millisecond, func() {
			win.PostMessage(p.hwnd, wmRequestRefresh, 0, 0)
			win.PostMessage(p.hwnd, win.WM_COMMAND, idRun, 0)
		})
		return
	}

	if installed && status == service.StatusRunning {
		p.serviceOp("Apply config", func() error {
			if err := p.service.Stop(); err != nil {
				return err
			}
			time.Sleep(500 * time.Millisecond)
			return p.service.Start()
		})
		return
	}

	p.appendLog("Config saved. Nothing is running, so no restart is needed.")
	p.refreshAsync()
}

func (p *controlPanel) serviceOp(name string, fn func() error) {
	p.appendLog("%s requested.", name)
	p.mu.Lock()
	p.refreshing = true
	p.mu.Unlock()
	p.applyButtons()
	go func() {
		if err := fn(); err != nil {
			p.appendLogAsync("%s failed: %v", name, err)
		} else {
			p.appendLogAsync("%s completed.", name)
		}
		p.mu.Lock()
		p.refreshing = false
		p.mu.Unlock()
		win.PostMessage(p.hwnd, wmRequestRefresh, 0, 0)
	}()
}

func (p *controlPanel) openConfigFolder() {
	path := p.configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		p.showError("Open config folder", err)
		return
	}
	if err := exec.Command("explorer.exe", filepath.Dir(path)).Start(); err != nil {
		p.showError("Open config folder", err)
	}
}

func (p *controlPanel) showError(title string, err error) {
	p.appendLog("%s failed: %v", title, err)
	messageBox(p.hwnd, title, err.Error(), win.MB_ICONERROR)
	p.refreshAsync()
}

func (p *controlPanel) openConfigDialog() {
	if currentConfig != nil {
		return
	}
	cfg, _, err := config.Load(p.configPath())
	if err != nil {
		p.showError("Open config", err)
		return
	}

	dlg := &configDialog{panel: controlPanelRef{p}}
	currentConfig = dlg
	hwnd := win.CreateWindowEx(
		win.WS_EX_DLGMODALFRAME,
		utf16Ptr(configClassName),
		utf16Ptr("Agent config"),
		win.WS_CAPTION|win.WS_SYSMENU,
		win.CW_USEDEFAULT,
		win.CW_USEDEFAULT,
		560,
		660,
		p.hwnd,
		0,
		hInstance,
		nil,
	)
	if hwnd == 0 {
		currentConfig = nil
		p.showError("Open config", fmt.Errorf("CreateWindowEx failed"))
		return
	}
	dlg.hwnd = hwnd
	dlg.load(cfg)
	enableWindow(p.hwnd, false)
	win.ShowWindow(hwnd, win.SW_SHOW)
	win.UpdateWindow(hwnd)
}

func (d *configDialog) createControls(hwnd win.HWND) {
	y := int32(16)
	d.label(hwnd, d.panel.configPath(), 16, y, 500, 22)
	y += 32

	d.portEdit = d.rowEdit(hwnd, "Port", y, idCfgPort)
	y += 28
	d.intervalEdit = d.rowEdit(hwnd, "Metrics interval ms", y, idCfgInterval)
	y += 32

	d.cpuCB = d.rowCheck(hwnd, "CPU", y, idCfgCPU)
	y += 24
	d.memoryCB = d.rowCheck(hwnd, "Memory", y, idCfgMemory)
	y += 24
	d.disksCB = d.rowCheck(hwnd, "Disks", y, idCfgDisks)
	y += 24
	d.networkCB = d.rowCheck(hwnd, "Network", y, idCfgNetwork)
	y += 24
	d.gpuCB = d.rowCheck(hwnd, "GPU", y, idCfgGPU)
	y += 24
	d.temperaturesCB = d.rowCheck(hwnd, "Temperatures", y, idCfgTemperatures)
	y += 24
	d.screensCB = d.rowCheck(hwnd, "Screens", y, idCfgScreens)
	y += 28

	d.interfaceCombo = d.rowCombo(hwnd, "Network interface", y, idCfgInterface)
	y += 28
	d.preferIPv4CB = d.rowCheck(hwnd, "Prefer IPv4", y, idCfgPreferIPv4)
	y += 28
	d.label(hwnd, "Disks", 16, y+2, 150, 22)
	d.diskChecks = d.createDiskChecks(hwnd, y)
	y += int32(max(1, len(d.diskChecks))) * 24
	d.imageDirEdit = d.rowEdit(hwnd, "Images directory", y, idCfgImageDir)
	y += 28
	d.changeEveryEdit = d.rowEdit(hwnd, "Change every seconds", y, idCfgChangeEvery)
	y += 28
	d.asciiWidthEdit = d.rowEdit(hwnd, "ASCII width", y, idCfgASCIIWidth)
	y += 28
	d.asciiHeightEdit = d.rowEdit(hwnd, "ASCII height", y, idCfgASCIIHeight)
	y += 28
	d.charsetEdit = d.rowEdit(hwnd, "Charset", y, idCfgCharset)
	y += 28
	d.paletteEdit = d.rowEdit(hwnd, "Palette size", y, idCfgPalette)

	d.button(hwnd, "Save", 340, 580, 80, 30, idCfgSave)
	d.button(hwnd, "Cancel", 430, 580, 80, 30, idCfgCancel)
}

func (d *configDialog) load(cfg *config.Config) {
	setWindowText(d.portEdit, strconv.Itoa(cfg.Server.Port))
	setWindowText(d.intervalEdit, strconv.Itoa(cfg.Metrics.IntervalMs))
	setChecked(d.cpuCB, cfg.Metrics.CPU)
	setChecked(d.memoryCB, cfg.Metrics.Memory)
	setChecked(d.disksCB, cfg.Metrics.Disks)
	setChecked(d.networkCB, cfg.Metrics.Network)
	setChecked(d.gpuCB, cfg.Metrics.GPU)
	setChecked(d.temperaturesCB, cfg.Metrics.Temperatures)
	setChecked(d.screensCB, cfg.Metrics.Screens)
	d.loadInterfaces(cfg.Network.InterfaceName)
	setChecked(d.preferIPv4CB, cfg.Network.PreferIPv4)
	d.loadDisks(cfg.Disks.Include)
	setWindowText(d.imageDirEdit, cfg.Images.Directory)
	setWindowText(d.changeEveryEdit, strconv.Itoa(cfg.Images.ChangeEverySeconds))
	setWindowText(d.asciiWidthEdit, strconv.Itoa(cfg.Images.ASCIIWidth))
	setWindowText(d.asciiHeightEdit, strconv.Itoa(cfg.Images.ASCIIHeight))
	setWindowText(d.charsetEdit, cfg.Images.Charset)
	setWindowText(d.paletteEdit, strconv.Itoa(cfg.Images.PaletteSize))
}

func (d *configDialog) loadInterfaces(selected string) {
	comboReset(d.interfaceCombo)
	comboAdd(d.interfaceCombo, "")
	selectedIndex := 0

	interfaces, err := metrics.Interfaces()
	if err != nil {
		if selected != "" {
			selectedIndex = comboAdd(d.interfaceCombo, selected)
		}
		comboSelect(d.interfaceCombo, selectedIndex)
		return
	}

	for _, nic := range interfaces {
		index := comboAdd(d.interfaceCombo, nic.Name)
		if strings.EqualFold(nic.Name, selected) {
			selectedIndex = index
		}
	}
	if selected != "" && selectedIndex == 0 {
		selectedIndex = comboAdd(d.interfaceCombo, selected)
	}
	comboSelect(d.interfaceCombo, selectedIndex)
}

func (d *configDialog) createDiskChecks(parent win.HWND, y int32) []diskCheck {
	volumes, err := metrics.Disks()
	if err != nil || len(volumes) == 0 {
		return []diskCheck{{
			name: "C:",
			hwnd: createControl(parent, "BUTTON", "C:", win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.BS_AUTOCHECKBOX, 0, 180, y, 60, 22, 0),
		}}
	}

	checks := make([]diskCheck, 0, len(volumes))
	for i, volume := range volumes {
		checks = append(checks, diskCheck{
			name: volume.Name,
			hwnd: createControl(parent, "BUTTON", volume.Name, win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.BS_AUTOCHECKBOX, 0, 180, y+int32(i*24), 120, 22, 0),
		})
	}
	return checks
}

func (d *configDialog) loadDisks(selected []string) {
	selectedSet := make(map[string]bool, len(selected))
	for _, name := range selected {
		selectedSet[strings.ToUpper(strings.TrimSpace(name))] = true
	}
	for _, disk := range d.diskChecks {
		setChecked(disk.hwnd, selectedSet[strings.ToUpper(disk.name)])
	}
}

func (d *configDialog) selectedDisks() []string {
	out := make([]string, 0, len(d.diskChecks))
	for _, disk := range d.diskChecks {
		if checked(disk.hwnd) {
			out = append(out, disk.name)
		}
	}
	return out
}

func (d *configDialog) handleCommand(id int) {
	switch id {
	case idCfgSave:
		d.save()
	case idCfgCancel:
		d.close()
	}
}

func (d *configDialog) save() {
	cfg, _, err := config.Load(d.panel.configPath())
	if err != nil {
		messageBox(d.hwnd, "Agent config", err.Error(), win.MB_ICONERROR)
		return
	}
	next, err := d.readConfig(cfg)
	if err != nil {
		messageBox(d.hwnd, "Agent config", err.Error(), win.MB_ICONERROR)
		return
	}
	if err := config.Save(d.panel.configPath(), next); err != nil {
		messageBox(d.hwnd, "Agent config", err.Error(), win.MB_ICONERROR)
		return
	}
	d.panel.appendLog("Config saved: %s", d.panel.configPath())
	d.close()
}

func (d *configDialog) readConfig(base *config.Config) (*config.Config, error) {
	port, err := parseRequiredInt("port", windowText(d.portEdit))
	if err != nil {
		return nil, err
	}
	interval, err := parseRequiredInt("metrics interval", windowText(d.intervalEdit))
	if err != nil {
		return nil, err
	}
	changeEvery, err := parseRequiredInt("change every seconds", windowText(d.changeEveryEdit))
	if err != nil {
		return nil, err
	}
	asciiWidth, err := parseRequiredInt("ASCII width", windowText(d.asciiWidthEdit))
	if err != nil {
		return nil, err
	}
	asciiHeight, err := parseRequiredInt("ASCII height", windowText(d.asciiHeightEdit))
	if err != nil {
		return nil, err
	}
	paletteSize, err := parseRequiredInt("palette size", windowText(d.paletteEdit))
	if err != nil {
		return nil, err
	}

	next := *base
	next.Server.Port = port
	next.Metrics.IntervalMs = interval
	next.Metrics.CPU = checked(d.cpuCB)
	next.Metrics.Memory = checked(d.memoryCB)
	next.Metrics.Disks = checked(d.disksCB)
	next.Metrics.Network = checked(d.networkCB)
	next.Metrics.GPU = checked(d.gpuCB)
	next.Metrics.Temperatures = checked(d.temperaturesCB)
	next.Metrics.Screens = checked(d.screensCB)
	next.Network.InterfaceName = selectedComboText(d.interfaceCombo)
	next.Network.PreferIPv4 = checked(d.preferIPv4CB)
	next.Disks.Include = d.selectedDisks()
	next.Images.Directory = strings.TrimSpace(windowText(d.imageDirEdit))
	next.Images.ChangeEverySeconds = changeEvery
	next.Images.ASCIIWidth = asciiWidth
	next.Images.ASCIIHeight = asciiHeight
	next.Images.Charset = windowText(d.charsetEdit)
	next.Images.PaletteSize = paletteSize
	next.Normalize()
	return &next, nil
}

func (d *configDialog) close() {
	enableWindow(d.panel.hwnd, true)
	win.DestroyWindow(d.hwnd)
	currentConfig = nil
}

func ensureConfigExists(path string) error {
	_, _, err := config.Load(path)
	return err
}

func parseRequiredInt(name, text string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func (p *controlPanel) label(parent win.HWND, text string, x, y, w, h int32) win.HWND {
	return createControl(parent, "STATIC", text, win.WS_CHILD|win.WS_VISIBLE, 0, x, y, w, h, 0)
}

func (p *controlPanel) edit(parent win.HWND, text string, x, y, w, h int32, id int) win.HWND {
	return createControl(parent, "EDIT", text, win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.ES_AUTOHSCROLL, win.WS_EX_CLIENTEDGE, x, y, w, h, id)
}

func (p *controlPanel) multilineEdit(parent win.HWND, text string, x, y, w, h int32, id int, readonly bool) win.HWND {
	style := uint32(win.WS_CHILD | win.WS_VISIBLE | win.WS_VSCROLL | win.ES_MULTILINE | win.ES_AUTOVSCROLL)
	if readonly {
		style |= win.ES_READONLY
	}
	return createControl(parent, "EDIT", text, style, win.WS_EX_CLIENTEDGE, x, y, w, h, id)
}

func (p *controlPanel) button(parent win.HWND, text string, x, y, w, h int32, id int) win.HWND {
	return createControl(parent, "BUTTON", text, win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.BS_PUSHBUTTON, 0, x, y, w, h, id)
}

func (d *configDialog) label(parent win.HWND, text string, x, y, w, h int32) win.HWND {
	return createControl(parent, "STATIC", text, win.WS_CHILD|win.WS_VISIBLE, 0, x, y, w, h, 0)
}

func (d *configDialog) button(parent win.HWND, text string, x, y, w, h int32, id int) win.HWND {
	return createControl(parent, "BUTTON", text, win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.BS_PUSHBUTTON, 0, x, y, w, h, id)
}

func (d *configDialog) rowEdit(parent win.HWND, label string, y int32, id int) win.HWND {
	d.label(parent, label, 16, y+3, 150, 22)
	return createControl(parent, "EDIT", "", win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.ES_AUTOHSCROLL, win.WS_EX_CLIENTEDGE, 180, y, 330, 24, id)
}

func (d *configDialog) rowCombo(parent win.HWND, label string, y int32, id int) win.HWND {
	d.label(parent, label, 16, y+3, 150, 22)
	return createControl(parent, "COMBOBOX", "", win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.CBS_DROPDOWNLIST|win.WS_VSCROLL, 0, 180, y, 330, 180, id)
}

func (d *configDialog) rowCheck(parent win.HWND, label string, y int32, id int) win.HWND {
	d.label(parent, label, 16, y+2, 150, 22)
	return createControl(parent, "BUTTON", "", win.WS_CHILD|win.WS_VISIBLE|win.WS_TABSTOP|win.BS_AUTOCHECKBOX, 0, 180, y, 24, 22, id)
}

func createControl(parent win.HWND, className, text string, style, exStyle uint32, x, y, w, h int32, id int) win.HWND {
	return win.CreateWindowEx(
		exStyle,
		utf16Ptr(className),
		utf16Ptr(text),
		style,
		x,
		y,
		w,
		h,
		parent,
		win.HMENU(uintptr(id)),
		hInstance,
		nil,
	)
}

func setWindowText(hwnd win.HWND, text string) {
	win.SendMessage(hwnd, win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(utf16Ptr(text))))
}

func windowText(hwnd win.HWND) string {
	length := int(win.SendMessage(hwnd, win.WM_GETTEXTLENGTH, 0, 0))
	buf := make([]uint16, length+1)
	win.SendMessage(hwnd, win.WM_GETTEXT, uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}

func appendEditText(hwnd win.HWND, text string) {
	length := win.SendMessage(hwnd, win.WM_GETTEXTLENGTH, 0, 0)
	win.SendMessage(hwnd, win.EM_SETSEL, length, length)
	win.SendMessage(hwnd, win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(utf16Ptr(text))))
}

func setChecked(hwnd win.HWND, value bool) {
	if value {
		win.SendMessage(hwnd, win.BM_SETCHECK, win.BST_CHECKED, 0)
		return
	}
	win.SendMessage(hwnd, win.BM_SETCHECK, win.BST_UNCHECKED, 0)
}

func checked(hwnd win.HWND) bool {
	return win.SendMessage(hwnd, win.BM_GETCHECK, 0, 0) == win.BST_CHECKED
}

func comboReset(hwnd win.HWND) {
	win.SendMessage(hwnd, win.CB_RESETCONTENT, 0, 0)
}

func comboAdd(hwnd win.HWND, text string) int {
	return int(win.SendMessage(hwnd, win.CB_ADDSTRING, 0, uintptr(unsafe.Pointer(utf16Ptr(text)))))
}

func comboSelect(hwnd win.HWND, index int) {
	win.SendMessage(hwnd, win.CB_SETCURSEL, uintptr(index), 0)
}

func selectedComboText(hwnd win.HWND) string {
	index := int(win.SendMessage(hwnd, win.CB_GETCURSEL, 0, 0))
	if index < 0 {
		return ""
	}
	length := int(win.SendMessage(hwnd, win.CB_GETLBTEXTLEN, uintptr(index), 0))
	if length <= 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	win.SendMessage(hwnd, win.CB_GETLBTEXT, uintptr(index), uintptr(unsafe.Pointer(&buf[0])))
	return strings.TrimSpace(syscall.UTF16ToString(buf))
}

func enableWindow(hwnd win.HWND, enabled bool) {
	win.EnableWindow(hwnd, enabled)
}

func messageBox(owner win.HWND, title, text string, flags uint32) {
	win.MessageBox(owner, utf16Ptr(text), utf16Ptr(title), flags)
}

func utf16Ptr(value string) *uint16 {
	return syscall.StringToUTF16Ptr(value)
}
