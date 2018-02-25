package grifts

import (
	"github.com/gobuffalo/buffalo"
	"github.com/UNO-SOFT/szamlazo/actions"
)

func init() {
	buffalo.Grifts(actions.App())
}
