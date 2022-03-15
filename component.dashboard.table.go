package main

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/kyoto-framework/kyoto"
)

type DashboardTable struct {
	Rows []DashboardTableRow
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

type DashboardTableRow struct {
	Ceg, User, Tipus, Summary              string
	ID                                     int32
	VallalasEnapCent, IdoEnapCent          EnapCent
	Beerkezes, Elfogadva, Hatarido, Elesen time.Time
}

func (dr *DashboardTableRow) ParseFields(row []string) error {
	if len(row) < 11 {
		return errors.New("short")
	}
	if len(row) >= 12 {
		row = row[1:12]
	}
	dr.Ceg, dr.User, dr.Tipus, dr.Summary = row[0], row[2], row[9], row[10]
	var firstErr error
	C := func(s string) EnapCent {
		i, err := strconv.ParseInt(strings.Replace(s, ",", "", 1), 10, 32)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%q: %w", s, err)
		}
		return EnapCent(i)
	}
	dr.VallalasEnapCent, dr.IdoEnapCent = C(row[3]), C(row[4])
	dr.ID = int32(C(row[1]))
	T := func(s string) time.Time {
		if s == "" {
			return time.Time{}
		}
		pattern := "2006-01-02 15:04:05"
		if len(s) == 10 {
			pattern = pattern[:10]
		}
		t, err := time.Parse(pattern, s)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%q: %w", s, err)
		}
		return t
	}
	dr.Beerkezes, dr.Elfogadva, dr.Hatarido, dr.Elesen = T(row[5]), T(row[6]), T(row[7]), T(row[8])
	return firstErr
}

type EnapCent int32

func (e EnapCent) String() string { return fmt.Sprintf("%d,%02d", e/100, e%100) }
