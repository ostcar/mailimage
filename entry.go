package main

import (
	"net/mail"
)

type Entry struct {
	address   *mail.Address
	subject   string
	text      string
	imageExt  string
	thumbnail []byte
}

type ServeEntry struct {
	ID        int    `json:"id"`
	From      string `json:"from"`
	Subject   string `json:"title"`
	Text      string `json:"text"`
	Extension string `json:"fileextension"`
	Created   string `json:"created"`
}

type ByCreated []ServeEntry

func (a ByCreated) Len() int           { return len(a) }
func (a ByCreated) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCreated) Less(i, j int) bool { return a[i].Created < a[j].Created }
