package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"golang.org/x/xerrors"
)

// mailimagePath returns the path where the mails and images are saved.
func mailimagePath() string {
	if os.Getenv("MAILIMAGE_PATH") == "" {
		return defaultMailimagePath
	}
	return os.Getenv("MAILIMAGE_PATH")
}

type mailFile struct {
	*os.File
	name   string
	folder string
}

// Create creates a new mailFile with the date as the name
func newMailFile() (*mailFile, error) {
	// Ensure folders
	if err := createFolders(); err != nil {
		return nil, xerrors.Errorf("can not create folders: %w", err)
	}

	f := mailFile{
		name:   fmt.Sprintf("%s-%02d.eml", time.Now().Format("2006-01-02_15-04-05"), rand.Intn(99)),
		folder: "progress",
	}

	fi, err := os.Create(f.path())
	if err != nil {
		return nil, xerrors.Errorf("can not open image file for writing: %w", err)
	}
	f.File = fi
	return &f, nil
}

// move moves the file from one folder to another
func (f *mailFile) move(folder string) error {
	if err := os.Rename(f.path(), path.Join(mailimagePath(), folder, f.name)); err != nil {
		return xerrors.Errorf("can not move mail from %s to %s: %w", f.folder, folder, err)
	}

	f.folder = folder
	return nil
}

// path returns the full path of the file (including the name of the file)
func (f *mailFile) path() string {
	return path.Join(mailimagePath(), f.folder, f.name)
}

// createFolders creates the folders in the filesystem to save the mails and the images
func createFolders() error {
	folders := [...]string{
		"progress",
		"error",
		"invalid",
		"success",
		"images",
	}
	for _, folder := range folders {
		if err := os.MkdirAll(path.Join(mailimagePath(), folder), os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
