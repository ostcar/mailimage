package main

import "os"

// mailimagePath returns the path where the mails and images are saved.
func mailimagePath() string {
	if os.Getenv("MAILIMAGE_PATH") == "" {
		return defaultMailimagePath
	}
	return os.Getenv("MAILIMAGE_PATH")
}
