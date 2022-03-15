package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/kyoto-framework/kyoto"
)

type DashboardTable struct {
	FilterDashboardTableRow
	Filtered []DashboardTableRow `json:"-"`
}

func (c *DashboardTable) Rows() []DashboardTableRow { return dashboardTableRows }

func (c *DashboardTable) Meta() kyoto.Meta {
	return kyoto.Meta{
		Title: "DashboardTable",
	}
}
func (c *DashboardTable) Template() *template.Template {
	return newtemplate("component.dashboard.table.html")
}
func (c *DashboardTable) Actions() kyoto.ActionMap {
	return kyoto.ActionMap{
		"Reset": func(args ...interface{}) {
			c.FilterDashboardTableRow = FilterDashboardTableRow{}
			c.Filtered = append(c.Filtered[:0], dashboardTableRows...)
		},
		"Reload": func(args ...interface{}) {
			c.Filtered = c.Filtered[:0]
			for _, item := range dashboardTableRows {
				if c.FilterDashboardTableRow.Match(item) {
					c.Filtered = append(c.Filtered, item)
				}
			}
		},
	}
}

func (c *DashboardTable) Len() int { return len(dashboardTableRows) }

type DashboardTableRow struct {
	Ceg, User, Tipus, Summary              string
	ID                                     int32
	Vallalas, Ido                          EnapCent
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
		i := strings.IndexByte(s, ',')
		if i < 0 {
			i = len(s)
			s += ",00"
		}
		p1, err := strconv.ParseInt(s[:i], 10, 32)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%q: %w", s, err)
		}
		p2, err := strconv.ParseInt(s[i+1:], 10, 32)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%q: %w", s, err)
		}
		return EnapCent(p1*100 + p2)
	}
	dr.Vallalas, dr.Ido = C(row[3]), C(row[4])
	dr.ID = int32(C(row[1])) / 100
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

type FilterDashboardTableRow struct {
	VallalasMin, VallalasMax   string
	IdoMin, IdoMax             string
	BeerkezesMin, BeerkezesMax Date
	ElfogadvaMin, ElfogadvaMax Date
	HataridoMin, HataridoMax   Date
	ElesenMin, ElesenMax       Date
	ID                         string
	DashboardTableRow
}

func (s FilterDashboardTableRow) Match(dr DashboardTableRow) bool {
	fc := func(s string) EnapCent {
		if s == "" {
			return 0
		}
		f, _ := strconv.ParseFloat(s, 32)
		return EnapCent(f * 100)
	}
	vallalasMin, vallalasMax := fc(s.VallalasMin), fc(s.VallalasMax)
	idoMin, idoMax := fc(s.IdoMin), fc(s.IdoMax)
	return (s.Ceg == "" || s.Ceg == dr.Ceg) &&
		(s.User == "" || s.User == dr.User) &&
		(s.Tipus == "" || s.Tipus == dr.Tipus) &&
		(s.ID == "" || s.ID == strconv.Itoa(int(dr.ID))) &&
		(vallalasMin == 0 && vallalasMax == 0 || vallalasMin <= dr.Vallalas && dr.Vallalas < vallalasMax) &&
		(idoMin == 0 && idoMax == 0 || idoMin <= dr.Ido && dr.Ido < idoMax) &&
		(s.BeerkezesMin.IsZero() && s.BeerkezesMax.IsZero() || !s.BeerkezesMin.After(dr.Beerkezes) && s.BeerkezesMax.After(dr.Beerkezes)) &&
		(s.ElfogadvaMin.IsZero() && s.ElfogadvaMax.IsZero() || !s.ElfogadvaMin.After(dr.Elfogadva) && s.ElfogadvaMax.After(dr.Elfogadva)) &&
		(s.HataridoMin.IsZero() && s.HataridoMax.IsZero() || !s.HataridoMin.After(dr.Hatarido) && s.HataridoMax.After(dr.Hatarido)) &&
		(s.ElesenMin.IsZero() && s.ElesenMax.IsZero() || !s.ElesenMin.After(dr.Elesen) && s.ElesenMax.After(dr.Elesen))
}

type EnapCent int32

func (e EnapCent) String() string { return fmt.Sprintf("%d,%02d", e/100, e%100) }

type Date struct {
	time.Time
}

func (d *Date) UnmarshalJSON(p []byte) error {
	var s string
	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}
	var err error
	if s == "" {
		d.Time = time.Time{}
	} else if len(s) == 10 {
		d.Time, err = time.Parse("2006-01-02", s)
	} else {
		d.Time, err = time.Parse(time.RFC3339, s)
	}
	return err
}
