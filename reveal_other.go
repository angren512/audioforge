//go:build !windows

package main

// revealInExplorer 在非 Windows 平台仅用于开发调试，直接忽略。
func revealInExplorer(string) {}
