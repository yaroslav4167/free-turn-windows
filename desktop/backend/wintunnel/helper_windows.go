//go:build windows

package wintunnel

import "errors"

func RunHelper(string) error { return errors.New("Linux VPN helper is unavailable on Windows") }
