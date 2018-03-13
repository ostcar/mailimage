package main

const (
	fromAdress        = "Baarfood <ernte@baarfood.de>"
	tokenLength       = 8
	tokenExpire       = 24 * 60 * 60
	baseURL           = "http://mailimage.oshahn.de"
	logPath           = "/var/log/mailimage.log"
	subjectLength     = 25
	textLength        = 200
	responseRegards   = "Viele Grüße\nBaarfood"
	deleteRedirectURL = "http://baarfood.de/ernte-teilen/"
)

var debug = false
