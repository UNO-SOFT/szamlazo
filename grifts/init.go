package grifts

import (
	"github.com/UNO-SOFT/szamlazo/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
