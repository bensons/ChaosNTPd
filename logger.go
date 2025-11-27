package main

import (
	"fmt"
	"time"
)

// Simple logging functions

func LogInfo(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] INFO: %s\n", timestamp, msg)
}

func LogWarning(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] WARN: %s\n", timestamp, msg)
}

func LogError(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] ERROR: %s\n", timestamp, msg)
}

func LogDebug(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] DEBUG: %s\n", timestamp, msg)
}
