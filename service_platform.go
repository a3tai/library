package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ToggleMaximise toggles the current window's maximised state. Wired to the
// title-bar double-click on the frontend so we mimic the macOS default
// "double-click to zoom" gesture even though the title bar is rendered by
// the webview.
func (s *LibraryService) ToggleMaximise() error {
	app := application.Get()
	if app == nil {
		return fmt.Errorf("application not initialised")
	}
	win := app.Window.Current()
	if win == nil {
		return fmt.Errorf("no current window")
	}
	win.ToggleMaximise()
	return nil
}

// PickImportPath opens the native file/folder picker and returns the
// chosen path. Returns an empty string (no error) when the user cancels.
//
// macOS 26 + Wails v3 alpha workaround: Wails' own NSOpenPanel path
// (attached or free-standing) consistently failed to surface a panel
// on macOS 26 Tahoe with Wails alpha.74. The C call returned but no
// panel ever appeared, leaving Go blocked on the response channel
// forever. We bypass that by default with `osascript`, which uses
// Finder's documented dialog scripting to present the picker.
//
// Set LIBRARY_PICKER=native to try the Wails NSOpenPanel path again —
// useful for verifying whether a Wails upgrade has fixed the regression.
// Anything else (or unset) keeps the osascript safety net.
func (s *LibraryService) PickImportPath() (string, error) {
	_, span := tracer().Start(context.Background(), "import.pick_path")
	defer span.End()
	if strings.EqualFold(firstEnv("LIBRARY_PICKER"), "native") {
		return s.pickImportPathNative()
	}
	log.Printf("[import] PickImportPath: called (osascript path)")

	// Bring the app to front so the picker doesn't open behind other
	// apps. Best-effort — failure is non-fatal.
	if app := application.Get(); app != nil {
		if win := app.Window.Current(); win != nil {
			win.Focus()
		} else if all := application.Get().Window.GetAll(); len(all) > 0 {
			all[0].Focus()
		}
	}

	// AppleScript: "choose file or folder" via the Finder. Returns a
	// POSIX path on selection; on cancel, osascript exits with code 1
	// and writes the error to stderr. We treat that as "no path" and
	// return an empty string.
	//
	// The `with prompt` text is shown above the panel. We deliberately
	// don't filter by extension here — the import pipeline already
	// skips unsupported files, and presenting a clean unfiltered picker
	// is friendlier than Wails' constrained one. (Users importing a
	// folder of mixed types still get the right behaviour.)
	const script = `set the source to choose file with prompt "Choose a file or a folder to add to your library." with multiple selections allowed of (false) without invisibles
return POSIX path of source`

	// Alternative script for folder selection. AppleScript doesn't have
	// a single "file or folder" picker in a single command, so we offer
	// the user the choice via a primary file picker, with a "Choose
	// Folder…" button that re-presents as `choose folder`.
	// In practice for a library import the user usually wants a folder.
	// Make the default a folder picker, with `choose file` available
	// via a fallback mode if we ever expose that toggle.
	const folderScript = `try
	set chosen to choose folder with prompt "Choose a folder to add to your library."
	return POSIX path of chosen
on error number -128
	return ""
end try`

	log.Printf("[import] PickImportPath: running osascript")
	cmd := exec.Command("osascript", "-e", folderScript)
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 on user cancel — AppleScript's "User cancelled"
		// error. Anything else is a real failure. Either way return ""
		// and let the frontend treat it as no-op.
		log.Printf("[import] PickImportPath: osascript exit err=%v", err)
		return "", nil
	}
	path := strings.TrimSpace(string(out))
	log.Printf("[import] PickImportPath: returned path=%q", path)
	_ = script // keep the file-picker variant around for future use
	return path, nil
}

// pickImportPathNative is the original Wails NSOpenPanel path, opt-in
// via LIBRARY_PICKER=native. Useful for verifying whether a Wails upgrade
// has fixed the macOS 26 "panel never surfaces" regression — if you
// click Import and a dialog appears within a couple of seconds, native
// is working again and we can drop the osascript fallback.
func (s *LibraryService) pickImportPathNative() (string, error) {
	log.Printf("[import] PickImportPath: called (native NSOpenPanel path)")
	app := application.Get()
	if app == nil {
		log.Printf("[import] PickImportPath: application not initialised")
		return "", fmt.Errorf("application not initialised")
	}

	dlg := app.Dialog.OpenFile().
		SetTitle("Import books").
		SetMessage("Choose a file or a folder to add to your library.").
		CanChooseFiles(true).
		CanChooseDirectories(true).
		ResolvesAliases(true).
		AddFilter("Supported books", "*.epub;*.pdf;*.txt")

	// Attach to the current window so the panel presents as a sheet,
	// which surfaces more reliably than free-standing modals on macOS
	// 26. Falls back to the first known window when Current() is nil.
	win := app.Window.Current()
	if win == nil {
		if all := app.Window.GetAll(); len(all) > 0 {
			win = all[0]
			log.Printf("[import] PickImportPath: Current()=nil, falling back to GetAll()[0]")
		}
	}
	if win != nil {
		win.Focus()
		dlg = dlg.AttachToWindow(win)
		log.Printf("[import] PickImportPath: dialog attached (window id=%d)", win.ID())
	} else {
		log.Printf("[import] PickImportPath: WARNING — no window available; dialog will run free-standing")
	}

	log.Printf("[import] PickImportPath: opening native picker")
	path, err := dlg.PromptForSingleSelection()
	log.Printf("[import] PickImportPath: returned path=%q err=%v", path, err)
	return path, err
}
