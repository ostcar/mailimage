package main

import (
	"fmt"

	"golang.org/x/xerrors"
)

var (
	errNoImage         = xerrors.New("Keine Bilddatei in der E-Mail gefunden.")
	errMultiImage      = xerrors.New("Mehrere Bilder gefunden. Die E-Mail darf maximal ein Bild entalten.")
	errParsingImage    = xerrors.New("Die Bilddatei kann nicht gelesen werden.")
	errCreateThumbnail = xerrors.New("Thumbnail kann aus dem Bild nicht erstellt werden.")
	errLongSubject     = xerrors.New(fmt.Sprintf("E-Mail Betreff ist zu lang. Maximal %d zeichen sind erlaubt.", subjectLength))
	errLongText        = xerrors.New(fmt.Sprintf("Text der E-Mail darf maximal %d Zeichen lang sein.", textLength))
	errInternal        = xerrors.New("Ups, etwas ist schief gelaufen. Bitte die Admins benachrichtigen.")
	errUnknownImage    = xerrors.New("Unknown image id")
)
