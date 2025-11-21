/*
   Espresso - A lightweight utility to keep your screen on and your system active.
   Copyright (C) 2025  Rodrigo Toraño Valle

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
)

//go:embed assets/icon.ico
var iconData []byte

//go:embed assets/icoff.ico
var icoffData []byte

//go:embed LICENSE
var licenseData []byte

//go:embed THIRD_PARTY_LICENSES
var thirdPartyLicenses embed.FS

var execStateCh = make(chan func())

// --- Constants & Config ---

const (
	defaultLanguage = "en-US"
)

var (
	modkernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procSetThreadExecutionState = modkernel32.NewProc("SetThreadExecutionState")
)

const (
	ES_CONTINUOUS       = 0x80000000
	ES_SYSTEM_REQUIRED  = 0x00000001
	ES_DISPLAY_REQUIRED = 0x00000002
	MB_ICONINFORMATION  = 0x00000040
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

type Config struct {
	Language string `json:"language"`
}

// --- Mode Definitions ---

type EspressoMode struct {
	Name     string
	Duration time.Duration // 0 for infinite
	Desc     string
}

var modes = []EspressoMode{
	{"Milk", 3 * time.Minute, "No caffeine, just for testing purposes."},
	{"Drop", 10 * time.Minute, "Just a drop, almost no caffeine."},
	{"Latte", 30 * time.Minute, "Gentle boost to get you started."},
	{"Cappuccino", 1 * time.Hour, "Noticeable caffeine, perfectly balanced."},
	{"Americano", 3 * time.Hour, "Stronger, long-lasting alertness."},
	{"Espresso", 6 * time.Hour, "Concentrated, powerful kick."},
	{"Lungo", 8 * time.Hour, "Super concentrated, extended energy."},
	{"Doppio", 12 * time.Hour, "Double espresso, full-on focus all day."},
	{"Pure Caffeine", -1, "Maximum alertness, use with caution."}, // -1 for infinite
}

// --- Sleep Control ---

func allowSleep() {
	procSetThreadExecutionState.Call(uintptr(ES_CONTINUOUS))
}

func preventSleep() {
	// Prevent system sleep and display sleep
	procSetThreadExecutionState.Call(uintptr(ES_CONTINUOUS | ES_SYSTEM_REQUIRED | ES_DISPLAY_REQUIRED))
}

// --- File System & Config ---

func settingsPath() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			appdata = dir
		} else {
			if cwd, err := os.Getwd(); err == nil {
				appdata = cwd
			} else {
				appdata = os.TempDir()
			}
		}
	}
	dir := filepath.Join(appdata, "Espresso")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Warning: could not create config dir %s: %v\n", dir, err)
	}
	return filepath.Join(dir, "settings.json")
}

func licenseFilePath() string {
	dir := filepath.Dir(settingsPath())
	return filepath.Join(dir, "LICENSE.txt")
}

func iconPath() string {
	dir := filepath.Dir(settingsPath())
	return filepath.Join(dir, "espresso.ico")
}

func icoffPath() string {
	dir := filepath.Dir(settingsPath())
	return filepath.Join(dir, "espressoff.ico")
}

func ensureResourceFiles() {
	// 1. Ensure License File
	p := licenseFilePath()
	diskData, err := os.ReadFile(p)
	writeRequired := false

	if os.IsNotExist(err) || err != nil || !bytes.Equal(diskData, licenseData) {
		writeRequired = true
	}

	if writeRequired {
		_ = os.WriteFile(p, licenseData, 0644)
	}

	// 2. Ensure Icon File (Required on disk for Toast notifications)
	p = iconPath()
	diskData, err = os.ReadFile(p)
	writeRequired = false

	if os.IsNotExist(err) || err != nil || !bytes.Equal(diskData, iconData) {
		writeRequired = true
	}

	if writeRequired {
		_ = os.WriteFile(p, iconData, 0644)
	}

	p = icoffPath()
	diskData, err = os.ReadFile(p)
	writeRequired = false

	if os.IsNotExist(err) || err != nil || !bytes.Equal(diskData, icoffData) {
		writeRequired = true
	}

	if writeRequired {
		_ = os.WriteFile(p, icoffData, 0644)
	}

	// 3. Ensure Third Party Licenses
	appDataDir := filepath.Dir(settingsPath())
	thirdPartyDir := filepath.Join(appDataDir, "THIRD_PARTY_LICENSES")
	_ = os.MkdirAll(thirdPartyDir, 0755)

	_ = fs.WalkDir(thirdPartyLicenses, "THIRD_PARTY_LICENSES", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(path, "THIRD_PARTY_LICENSES/")
		destPath := filepath.Join(thirdPartyDir, relPath)

		embeddedContent, _ := thirdPartyLicenses.ReadFile(path)
		diskContent, diskErr := os.ReadFile(destPath)

		if os.IsNotExist(diskErr) || !bytes.Equal(diskContent, embeddedContent) {
			_ = os.WriteFile(destPath, embeddedContent, 0644)
		}
		return nil
	})
}

func loadConfig() Config {
	defaultCfg := Config{
		Language: defaultLanguage,
	}

	p := settingsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		_ = saveConfig(defaultCfg)
		return defaultCfg
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		_ = saveConfig(defaultCfg)
		return defaultCfg
	}

	needsSave := false

	if cfg.Language == "" {
		cfg.Language = defaultLanguage
		needsSave = true
	}

	if needsSave {
		saveConfig(cfg)
	}

	return cfg
}

func saveConfig(cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(settingsPath(), data, 0644)
}

// --- UI Helpers ---

func showMessage(title, message string) {
	t, _ := windows.UTF16PtrFromString(title)
	m, _ := windows.UTF16PtrFromString(message)

	procMessageBoxW.Call(0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(MB_ICONINFORMATION))
}

func showToast(title, message string, iconPath string) error {
	notification := toast.Notification{
		AppID:   "Espresso",
		Title:   title,
		Message: message,
		Icon:    iconPath,
		// Audio:    toast.IM,
		Duration: toast.Short,
		Actions: []toast.Action{
			{Type: "protocol", Label: "OK", Arguments: ""},
		},
	}

	err := notification.Push()
	if err != nil {
		fmt.Printf("Error showing toast notification: %v\n", err)
	}
	return err
}

func showAbout() {
	mainLicensePath := licenseFilePath()
	thirdPartyLicensesDir := filepath.Join(filepath.Dir(settingsPath()), "THIRD_PARTY_LICENSES")

	aboutMessage := fmt.Sprintf(
		"Espresso - A lightweight utility to keep your screen on and your system active.\n\n"+
			"Copyright (C) 2025  Rodrigo Toraño Valle\n\n"+
			"This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.\n\n"+
			"This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.\n\n"+
			"You should have received a copy of the GNU General Public License along with this program.  If not, see <https://www.gnu.org/licenses/>.\n\n"+
			"You can find the full GPLv3 license text in:\n%s\n\n"+
			"Required notices for third-party components (Apache-2.0, BSD-3-Clause) are located in the following folder:\n%s\n\n\n\n",
		mainLicensePath,
		thirdPartyLicensesDir,
	)
	showMessage("About Espresso", aboutMessage)
	// go func() {
	// 	_ = exec.Command("notepad", mainLicensePath).Start()
	// }()
}

// --- Main Execution ---

var instanceMutex windows.Handle

func enforceSingleInstance() bool {
	const mutexName = "Global\\EspressoAppMutex"
	h, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err != nil {
		return false
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(h)
		return false
	}
	instanceMutex = h
	return true
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	totalSeconds := int(d.Seconds())
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	h := totalSeconds / 3600
	m := (totalSeconds / 60) % 60
	s := totalSeconds % 60
	return fmt.Sprintf("%dh %dm %ds", h, m, s)
}

func formatFriendlyDuration(d time.Duration) string {
	if d < 0 {
		return "Infinity"
	}
	if d >= time.Hour && d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

// execOnMainThread ensures the Windows API call happens on the locked OS thread
func execOnMainThread(fn func()) {
	done := make(chan struct{})
	execStateCh <- func() {
		fn()
		close(done)
	}
	<-done
}

func startExecThread() {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		for fn := range execStateCh {
			if fn != nil {
				fn()
			}
		}
	}()
}

func main() {
	startExecThread()
	if !enforceSingleInstance() {
		return
	}
	// Ensure we start allowing sleep
	execOnMainThread(func() { allowSleep() })
	systray.Run(onReady, onExit)
}

func onReady() {
	ensureResourceFiles()
	systray.SetIcon(icoffData)
	systray.SetTitle("Espresso")
	systray.SetTooltip("Espresso: Decaf (Sleep allowed)")

	cfg := loadConfig()
	fmt.Printf("Loaded config: %+v\n", cfg)

	// --- Menu Items ---
	mInfo := systray.AddMenuItem("About Espresso", "Show info")
	systray.AddSeparator()

	mMode := systray.AddMenuItem("Mode: Decaf", "Current mode")
	mMode.Disable()

	mTimeLeft := systray.AddMenuItem("", "")
	mTimeLeft.Hide()

	systray.AddSeparator()

	// --- Dynamic Menu Creation ---
	controlCh := make(chan EspressoMode)

	for _, mode := range modes {
		label := fmt.Sprintf("%s (%s)", mode.Name, formatFriendlyDuration(mode.Duration))
		item := systray.AddMenuItem(label, mode.Desc)
		go func() {
			for range item.ClickedCh {
				controlCh <- mode
			}
		}()
	}

	systray.AddSeparator()
	mStop := systray.AddMenuItem("Decaf (Stop)", "Allow computer to sleep")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit program")

	// --- State Variables ---
	var (
		isActive        bool
		isInfinite      bool
		sessionEndTime  time.Time
		currentModeName string
	)

	resetState := func() {
		isActive = false
		isInfinite = false
		currentModeName = "Decaf"

		// System Call: Allow Sleep
		execOnMainThread(func() { allowSleep() })

		// Update UI
		systray.SetIcon(icoffData)
		mMode.SetTitle("Mode: Decaf")
		systray.SetTooltip("Espresso: Decaf (Sleep allowed)")
		mTimeLeft.Hide()
	}

	startSession := func(d time.Duration) {
		isActive = true

		// Determine name based on duration
		foundName := "Custom"
		for _, m := range modes {
			if m.Duration == d {
				foundName = m.Name
				break
			}
		}
		currentModeName = foundName

		// System Call: Prevent Sleep
		execOnMainThread(func() { preventSleep() })

		systray.SetIcon(iconData)

		if d < 0 {
			isInfinite = true
			mMode.SetTitle(fmt.Sprintf("Mode: %s (Infinite)", foundName))
			mTimeLeft.Hide()
			systray.SetTooltip("Espresso: Caffeine High (No Sleep)")
		} else {
			isInfinite = false
			sessionEndTime = time.Now().Add(d)
			mMode.SetTitle(fmt.Sprintf("Mode: %s (%s)", foundName, formatFriendlyDuration(d)))
			mTimeLeft.Show()
		}
	}

	// --- Main Loop ---
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-mInfo.ClickedCh:
				go showAbout()

			case <-mQuit.ClickedCh:
				systray.Quit()
				return

			case <-mStop.ClickedCh:
				resetState()
				showToast("Espresso Stopped", "System is now allowed to sleep", icoffPath())

			case m := <-controlCh:
				d := m.Duration
				startSession(d)
				var durationText string
				if d < 0 {
					durationText = "Preventing sleep indefinitely"
				} else {
					durationText = fmt.Sprintf("%s\nPreventing sleep for %s", m.Desc, formatFriendlyDuration(d))
				}
				showToast(fmt.Sprintf("%s Mode Started", m.Name), durationText, iconPath())

			case <-ticker.C:
				if !isActive {
					continue
				}

				if isInfinite {
					continue
				}

				remaining := time.Until(sessionEndTime)

				if remaining <= 0 {
					// Time is up!
					resetState()

					// Notify User
					go func() {
						showToast("Espresso Finished", "System is now allowed to sleep", icoffPath())
					}()
				} else {
					// Update UI Countdown
					timeStr := formatDuration(remaining)
					mTimeLeft.SetTitle(fmt.Sprintf("Time left: %s", timeStr))
					systray.SetTooltip(fmt.Sprintf("%s mode: %s remaining", currentModeName, timeStr))
				}
			}
		}
	}()
}

func onExit() {
	if instanceMutex != 0 {
		_ = windows.CloseHandle(instanceMutex)
		instanceMutex = 0
	}
	close(execStateCh)
}
