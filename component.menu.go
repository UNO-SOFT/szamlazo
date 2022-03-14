package main

import "html/template"

type Menu struct {
	Links []Link
}

type Link struct {
	Icon        template.HTML
	Title       string
	Description string
	Href        string
}
