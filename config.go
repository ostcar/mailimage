package main

const (
	redisAddr = ":6379"
	smptHost  = "localhost"
	smptPort  = 25

	tokenLength          = 8
	tokenExpire          = 24 * 60 * 60
	defaultLogPath       = "/var/log/mailimage.log"
	defaultMailimagePath = "/srv/mailimage/mails"
	subjectLength        = 25
	textLength           = 200
	baseURL              = "https://ernte.baarfood.de"
	deleteRedirectURL    = "https://baarfood.de/ernte-teilen/"

	fromAdress      = "Baarfood <ernte@baarfood.de>"
	responseRegards = "Viele Grüße\nBaarfood"
)

const sendErrorTemplate = `Hallo{{ if .Name }} {{ .Name }}{{ end }},

dein Bild kann nicht gespeichert werden. Bitte behebe {{ $length := len .Errors }}{{ if gt $length 1 }}den folgenden{{ else }}die folgenden{{ end }} Fehler:
{{ range .Errors }}* {{ . }}
{{ end }}

{{ .Regards }}
`

const sendSuccessTemplate = `Hallo {{ .Name }},

dein Bild wurde erfolgreich veröffentlicht. Über folgenden Link kannst du es
aufrufen:

{{ .ImageLink }}

In den folgenden 24 Stunden kannst du es mit einem klick auf den folgenden Link
wieder löschen:

{{ .RemoveLink }}

{{ .Regards }}
`

var allowedFormats = [...]string{
	"jpeg",
	"png",
	"jpg",
}
