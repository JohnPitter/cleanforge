package main

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=1.2.0"
var Version = "1.1.0"

func (a *App) GetVersion() string {
	return Version
}
