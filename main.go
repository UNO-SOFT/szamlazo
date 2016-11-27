package main

import (
	"flag"
	"log"
	"os"

	"github.com/icza/gowut/gwu"
)

func main() {
	flagTLSCert := flag.String("tls-cert", "", "TLS certificate file")
	flagTLSKey := flag.String("tls-key", "", "TLS key file")
	flag.Parse()
	addr := flag.Arg(0)
	if addr == "" {
		addr = "localhost:8081"
	}
	if err := Main(addr, *flagTLSCert, *flagTLSKey); err != nil {
		log.Fatalf("%+v", err)
	}
}
func Main(addr, tlsCert, tlsKey string) error {
	// And to auto-create sessions for the login window:
	var server gwu.Server
	if tlsKey == "" || tlsCert == "" {
		server = gwu.NewServer("Számlázó", addr)
	} else {
		server = gwu.NewServerTLS("Számlázó", addr, tlsCert, tlsKey)
	}

	server.AddSessCreatorName("login", "Bejelentkezés")
	server.AddSHandler(sessHandler{})
	server.SetLogger(log.New(os.Stderr, "[szamlazo] ", log.LstdFlags))

	server.AddWin(mainWindow())
	return server.Start("")
}

func mainWindow() gwu.Window {
	// Create and build a window
	win := gwu.NewWindow("main", "Test GUI Window")
	win.Style().SetFullWidth()
	win.SetHAlign(gwu.HACenter)
	win.SetCellPadding(2)
	return win
}

// A SessionHandler implementation:
type sessHandler struct{}

func (h sessHandler) Created(s gwu.Session) {
	//win := gwu.NewWindow("login", "Bejelentkezés")
	// ...add content to the login window...
	//h.AddWindow(win)
}
func (h sessHandler) Removed(s gwu.Session) {}
