// asusimemode: get or force the Chinese/English mode of ASUS Bopomofo (華碩注音)
// in the foreground window.
//
// Usage:
//
//	asusimemode         print "zh" or "en"
//	asusimemode zh      force Chinese mode
//	asusimemode en      force English mode
//	asusimemode toggle  toggle mode
//
// ASUS Bopomofo ignores the IMM conversion mode. Its real state is written to
// [DirectOutputMode] in its per-user AsusIMEConfig.ini on every switch, keyed
// by process name (0 = Chinese, 1 = English), and any change of the IMM open
// status flips the mode. So we read the ini and flip the open status when the
// mode differs from the requested one.
//
// Switching needs an input context on the focused control. Text inputs always
// have one; other controls may or may not, depending on the program. A control
// without one gets no open status, so nothing can be switched, and moving the
// focus there changes the open status and thereby flips the mode: the state
// read then says nothing about the mode the IME restores on returning to an
// input.
//
// All Chinese IMEs share the same keyboard layout language, so the active one
// can't be told apart from outside its process. We refuse to run when any
// other Chinese IME is enabled.
//
// `task build` builds asusimemode.exe for shells and asusimemodew.exe (GUI
// subsystem) for hotkeys, so launching it from a hotkey doesn't open a console
// that steals focus.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	asusImeClsid = "{D9012138-88BF-46F4-B599-E69D8BA2FED6}"

	wmImeControl                   = 0x0283
	imcGetOpenStatus               = 0x0005
	imcSetOpenStatus               = 0x0006
	langChinese                    = 0x04
	processQueryLimitedInformation = 0x1000

	exitUsage              = 2
	exitNoIme              = 3
	exitNotChineseKeyboard = 4
	exitSwitchFailed       = 5
	exitUnsupported        = 6
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	imm32                          = syscall.NewLazyDLL("imm32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procGetKeyboardLayout          = user32.NewProc("GetKeyboardLayout")
	procSendMessageW               = user32.NewProc("SendMessageW")
	procImmGetDefaultIMEWnd        = imm32.NewProc("ImmGetDefaultIMEWnd")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

func main() {
	action := ""
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	if action != "" && action != "zh" && action != "en" && action != "toggle" {
		name := filepath.Base(os.Args[0])
		fmt.Fprintf(
			os.Stderr,
			"usage: %s [zh|en|toggle]\n",
			strings.TrimSuffix(name, filepath.Ext(name)),
		)
		os.Exit(exitUsage)
	}

	err := checkOnlyAsusChineseIme()
	if err != nil {
		fail(exitUnsupported, err)
	}

	fg, _, _ := procGetForegroundWindow.Call()
	imeWnd, _, _ := procImmGetDefaultIMEWnd.Call(fg)
	if fg == 0 || imeWnd == 0 {
		fail(exitNoIme, errors.New("no IME window for foreground window"))
	}

	var pid uint32
	tid, _, _ := procGetWindowThreadProcessId.Call(fg, uintptr(unsafe.Pointer(&pid)))
	hkl, _, _ := procGetKeyboardLayout.Call(tid)
	if hkl&0x3FF != langChinese {
		if action == "" {
			fmt.Println("en")
			return
		}
		fail(
			exitNotChineseKeyboard,
			fmt.Errorf("active keyboard %04X is not ASUS Bopomofo", hkl&0xFFFF),
		)
	}

	exe, err := processName(pid)
	if err != nil {
		fail(exitUnsupported, err)
	}
	zh, err := isChineseMode(exe)
	if err != nil {
		fail(exitUnsupported, err)
	}

	want := zh
	switch action {
	case "zh":
		want = true
	case "en":
		want = false
	case "toggle":
		want = !zh
	}

	if want != zh {
		open, _, _ := procSendMessageW.Call(imeWnd, wmImeControl, imcGetOpenStatus, 0)
		procSendMessageW.Call(imeWnd, wmImeControl, imcSetOpenStatus, open^1)
		// The IME writes the new mode to its ini asynchronously.
		for deadline := time.Now().Add(500 * time.Millisecond); zh != want && time.Now().Before(deadline); {
			time.Sleep(20 * time.Millisecond)
			zh, err = isChineseMode(exe)
			if err != nil {
				fail(exitUnsupported, err)
			}
		}
	}

	if zh {
		fmt.Println("zh")
	} else {
		fmt.Println("en")
	}
	if zh != want {
		// There is no way to check for an input context from outside the
		// process, so this is the only sign of a focused control without one.
		// Moving the focus to a text input is the fix the user can make.
		fail(exitSwitchFailed, errors.New("mode did not change; is the focus in a text input?"))
	}
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(code)
}

// checkOnlyAsusChineseIme verifies that every enabled Chinese input method is
// ASUS Bopomofo. Entries under each language look like
// "0404:{CLSID}{PROFILE}".
func checkOnlyAsusChineseIme() error {
	const profilePath = `Control Panel\International\User Profile`
	root, err := registry.OpenKey(registry.CURRENT_USER, profilePath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return fmt.Errorf("read input methods: %w", err)
	}
	defer root.Close()
	langs, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("read input methods: %w", err)
	}

	found := false
	for _, lang := range langs {
		k, err := registry.OpenKey(
			registry.CURRENT_USER,
			profilePath+`\`+lang,
			registry.QUERY_VALUE,
		)
		if err != nil {
			continue
		}
		names, _ := k.ReadValueNames(-1)
		k.Close()
		for _, name := range names {
			langID, tip, ok := strings.Cut(name, ":")
			id, err := strconv.ParseUint(langID, 16, 16)
			if !ok || err != nil || id&0x3FF != langChinese {
				continue
			}
			if !strings.HasPrefix(strings.ToUpper(tip), asusImeClsid) {
				return fmt.Errorf(
					"another Chinese input method is enabled (%s); can't tell which one is active",
					name,
				)
			}
			found = true
		}
	}
	if !found {
		return errors.New("ASUS Bopomofo is not enabled")
	}
	return nil
}

func processName(pid uint32) (string, error) {
	h, _, err := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return "", fmt.Errorf("open foreground process: %w", err)
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, syscall.MAX_LONG_PATH)
	size := uint32(len(buf))
	if r, _, err := procQueryFullProcessImageNameW.Call(
		h,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	); r == 0 {
		return "", fmt.Errorf("query foreground process name: %w", err)
	}
	return filepath.Base(syscall.UTF16ToString(buf[:size])), nil
}

// isChineseMode reads the IME's last recorded mode for exe. Apps without an
// entry start in Chinese mode.
func isChineseMode(exe string) (bool, error) {
	path := filepath.Join(
		os.Getenv("ProgramFiles"),
		`ASUS\AsusIME\zhtw\data`,
		os.Getenv("USERNAME"),
		"AsusIMEConfig.ini",
	)
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("read ASUS IME state: %w", err)
	}
	defer f.Close()

	inSection := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inSection = strings.EqualFold(line, "[DirectOutputMode]")
			continue
		}
		if k, v, ok := strings.Cut(line, "="); inSection && ok && strings.EqualFold(k, exe) {
			return v != "1", nil
		}
	}
	return true, sc.Err()
}
