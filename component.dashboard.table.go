package main

import (
	"html/template"

	"github.com/kyoto-framework/kyoto"
)

type DashboardTable struct {
	Rows [][]string
}

func (p *DashboardTable) Meta() kyoto.Meta {
	return kyoto.Meta{
		Title: "DashboardTable",
	}
}
func (p *DashboardTable) Template() *template.Template {
	return newtemplate("component.dashboard.table.html")
}
func (p *DashboardTable) Len() int { return len(p.Rows) }
