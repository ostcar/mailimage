package main

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/disintegration/imaging"
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
	if err := os.MkdirAll(path.Join(mailimagePath(), folder), os.ModePerm); err != nil {
		return xerrors.Errorf("can not create folder %s: %w", folder, err)
	}

	if err := os.Rename(f.path(), path.Join(mailimagePath(), folder, f.name)); err != nil {
		return xerrors.Errorf("can not move mail from %s to %s: %w", f.folder, folder, err)
	}

	f.folder = folder
	return nil
}

// remove changes the name of the file.
func (f *mailFile) rename(name string) error {
	if err := os.Rename(f.path(), path.Join(mailimagePath(), f.folder, name)); err != nil {
		return xerrors.Errorf("can not rename mail from %s to %s: %w", f.name, name, err)
	}

	f.name = name
	return nil
}

// path returns the full path of the file (including the name of the file)
func (f *mailFile) path() string {
	return path.Join(mailimagePath(), f.folder, f.name)
}

func openImage(id int, ext string) (*os.File, error) {
	path := path.Join(mailimagePath(), "images", fmt.Sprintf("%d%s", id, ext))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errUnknownImage
		}
		return nil, xerrors.Errorf("can not open image %d%s: %s", id, ext, err)
	}
	return f, nil
}

func openThumbnail(id int, redis *pool) (*os.File, error) {
	path := path.Join(mailimagePath(), "thumbnail", fmt.Sprintf("%d.jpg", id))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := createThumbnail(id, redis); err != nil {
				return nil, err
			}
			return openThumbnail(id, redis)
		}
		return nil, xerrors.Errorf("can not open thumbnail: %w", err)
	}
	return f, nil
}

func createThumbnail(id int, redis *pool) error {
	ext, err := redis.getExtension(id)
	if err != nil {
		return err
	}

	f, err := openImage(id, ext)
	if err != nil {
		return xerrors.Errorf("can not open image %s%ext: %w", id, ext, err)
	}

	image, err := imaging.Decode(f)
	if err != nil {
		f.Close()
		return xerrors.Errorf("can not read image: %w", err)
	}
	f.Close()

	// scale image
	image = imaging.Fill(image, 250, 200, imaging.Center, imaging.Lanczos)

	if err := os.MkdirAll(path.Join(mailimagePath(), "thumbnail"), os.ModePerm); err != nil {
		return xerrors.Errorf("can not create folder thumbnail: %w", err)
	}

	f, err = os.Create(path.Join(mailimagePath(), "thumbnail", fmt.Sprintf("%d.jpg", id)))
	if err != nil {
		return xerrors.Errorf("can not create thumbnail: %w", err)
	}
	defer f.Close()

	if err := imaging.Encode(f, image, imaging.JPEG); err != nil {
		return xerrors.Errorf("can not write thumbnail: %w", err)
	}
	return nil
}
