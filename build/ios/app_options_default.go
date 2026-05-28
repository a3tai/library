//go:build !ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS is a no-op on non-iOS platforms
func modifyOptionsForIOS(opts *application.Options) {
	// No modifications needed for non-iOS platforms
}

// main lets generic desktop-host checks such as `go build ./...` traverse
// this iOS scaffold package. Real iOS builds use main_ios.go.
func main() {}
