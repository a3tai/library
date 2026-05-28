package main

import (
	"context"
	"embed"
	_ "embed"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/a3tai/library/internal/windowstate"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// OTel tracer provider. No-op when neither LIBRARY_OTLP_ENDPOINT nor
	// LIBRARY_TRACE_STDOUT is set, so this is free for users who don't
	// want tracing. Shutdown is deferred so the last batch of spans
	// flushes before the process exits.
	shutdownTracing := initTracing(context.Background())
	defer shutdownTracing()

	libraryService, err := NewLibraryService("")
	if err != nil {
		log.Fatal(err)
	}

	app := application.New(application.Options{
		Name:        "A3T: Library",
		Description: "Index and browse a local book library",
		Services: []application.Service{
			application.NewService(libraryService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	winOpts := application.WebviewWindowOptions{
		Title: "A3T: Library",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			// LiquidGlass backdrop on macOS 15+ (Tahoe). Wails falls back
			// to MacBackdropTranslucent on older systems automatically.
			Backdrop: application.MacBackdropLiquidGlass,
			TitleBar: application.MacTitleBarHiddenInset,
			LiquidGlass: application.MacLiquidGlass{
				Style: application.LiquidGlassStyleAutomatic,
				// Match the Tahoe system corner radius (~16pt for windows
				// with a toolbar). Keep the HTML body clipped to this same
				// radius (see frontend/src/app.css) so the webview content
				// doesn't paint past the OS-rounded edge.
				CornerRadius: 16,
			},
		},
		// Transparent background so the OS-rounded corners aren't filled
		// with a solid color in the brief moment before the webview paints.
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/",
	}

	// Restore prior geometry if we have something plausible saved.
	saved, _ := windowstate.Load()
	if saved.Valid() {
		winOpts.Width = saved.Width
		winOpts.Height = saved.Height
		winOpts.X = saved.X
		winOpts.Y = saved.Y
		winOpts.InitialPosition = application.WindowXY
	}

	win := app.Window.NewWithOptions(winOpts)

	// Install the application menu AFTER the window is created. macOS
	// convention is to create windows first so the menu binds to an
	// existing app/window context; setting the menu before any window
	// exists has been observed to leave the app in a state where modal
	// panels (NSOpenPanel) silently fail to surface on macOS 26.
	app.Menu.Set(buildMenu(app))

	if saved.Maximised {
		// Defer one tick so the window is fully created before we tell macOS to zoom it.
		go func() {
			time.Sleep(150 * time.Millisecond)
			win.Maximise()
		}()
	}

	// Persist window geometry on a 1.5s tick whenever it changes. Wails v3's
	// resize/move event names move between alphas; polling is cheap and
	// guaranteed to work across platforms.
	stopWatch := make(chan struct{})
	go watchWindowGeometry(win, saved, stopWatch)

	err = app.Run()
	close(stopWatch)
	// One last save on clean shutdown so we capture the final position even
	// if the user closed the app between ticks.
	saveCurrentGeometry(win)

	if err != nil {
		log.Fatal(err)
	}
}

// buildMenu wires the macOS application menu. We keep the standard system
// roles (about, services, hide, quit) and add a Settings… item under the
// app submenu with the conventional ⌘, accelerator. Clicking the item
// emits a `app:open-settings` event that the frontend listens for to
// switch to the settings view; sidebar navigation hits the same view via
// a different path.
func buildMenu(app *application.App) *application.Menu {
	menu := app.Menu.New()

	// App menu. AddRole(AppMenu) gives the standard items; we then
	// reach into the submenu to insert a Settings… entry above Quit. The
	// AppMenu role builds the menu lazily, so we build a custom submenu
	// here that mirrors the system layout but lets us drop our item in.
	appMenu := menu.AddSubmenu("A3T: Library")
	appMenu.Add("About A3T: Library").OnClick(func(*application.Context) {
		app.Menu.ShowAbout()
	})
	appMenu.AddSeparator()
	appMenu.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) {
		app.Event.Emit("app:open-settings")
	})
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.UnHide)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Quit)

	// Standard Edit / View / Window / Help so the OS-provided keybindings
	// (copy/paste, minimise/zoom, etc.) work as expected.
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.ViewMenu)
	menu.AddRole(application.WindowMenu)
	menu.AddRole(application.HelpMenu)

	return menu
}

func watchWindowGeometry(win *application.WebviewWindow, last windowstate.State, stop <-chan struct{}) {
	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			cur := readGeometry(win)
			if !cur.Valid() {
				continue
			}
			// Don't overwrite saved coords while maximised — restoring would
			// dump the user back into a tiny window at the maximised origin.
			if cur.Maximised && last.Maximised {
				continue
			}
			if cur.Equal(last) {
				continue
			}
			if err := windowstate.Save(cur); err == nil {
				last = cur
			}
		}
	}
}

func readGeometry(win *application.WebviewWindow) windowstate.State {
	if win == nil {
		return windowstate.State{}
	}
	x, y := win.Position()
	w, h := win.Size()
	return windowstate.State{
		X:         x,
		Y:         y,
		Width:     w,
		Height:    h,
		Maximised: win.IsMaximised(),
	}
}

func saveCurrentGeometry(win *application.WebviewWindow) {
	if win == nil {
		return
	}
	cur := readGeometry(win)
	if cur.Valid() {
		_ = windowstate.Save(cur)
	}
}
