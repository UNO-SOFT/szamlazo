package main

import (
	"html/template"

	"github.com/kyoto-framework/kyoto"
)

type Dashboard struct {
	Navbar kyoto.Component
	Table  kyoto.Component
}

func (p *Dashboard) Template() *template.Template {
	return newtemplate("page.dashboard.html")
}

func (p *Dashboard) Meta() kyoto.Meta {
	return kyoto.Meta{
		Title: "Dashboard",
	}
}

func (p *Dashboard) Init() {
	p.Navbar = kyoto.RegC(p, &UINavbar)
	p.Table = kyoto.RegC(p, &DashboardTable{Rows: dashboardTableRows})
	//p.Menu = kyoto.RegC(p, &Menu{})
}
