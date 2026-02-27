package main

import "strings"

// Version is set at build time via ldflags:
//
//	go build -ldflags "-X main.Version=v1.2.0"
var Version = "1.4.0"

func (a *App) GetVersion() string {
	return strings.TrimPrefix(Version, "v")
}
