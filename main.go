package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/UNO-SOFT/szamlazo/oidcauth"
	"github.com/icza/gowut/gwu"
	"github.com/lucas-clemente/quic-go/h2quic"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var clientID, clientSecret, msTenantID string

func main() {
	flagTLSCert := flag.String("tls-cert", "", "TLS certificate file")
	flagTLSKey := flag.String("tls-key", "", "TLS key file")
	flag.StringVar(&clientID, "client-id", os.Getenv("CLIENT_ID"), "client ID")
	flag.StringVar(&clientSecret, "client-secret", os.Getenv("CLIENT_SECRET"), "client Secret")
	flag.Parse()

	if err := oidcauth.RegisterProvider(context.Background(),
		"Google", "", clientID, clientSecret,
		"profile",
	); err != nil {
		log.Fatal(err)
	}
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

	baseURL := server.AppUrl()
	baseURL = baseURL[:len(baseURL)-len(server.AppPath())]
	server.AddSessCreatorName("login", "Login Window")
	sH := sessHandler{destURL: server.AppUrl() + "main"}
	server.AddSHandler(sH)
	server.SetLogger(log.New(os.Stderr, "[szamlazo] ", log.LstdFlags))
	oidcauth.RegisterCbHandlers(context.Background(), http.DefaultServeMux,
		baseURL+"/_auth/{{lc .Provider}}/callback", sH.destURL,
		func(sess gwu.Session) { sess.RemoveWin(sess.WinByName("login")) },
	)

	if tlsCert == "" || tlsKey == "" {
		return server.Start("")
	}
	go server.Start("")
	return h2quic.ListenAndServeQUIC(addr, tlsCert, tlsKey, nil)
}

func loginWindow(ctx context.Context, destURL string) gwu.Window {
	if clientID == "" || clientSecret == "" {
		log.Println("Empty Client ID/Secret means no auth will work!")
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}

	win := gwu.NewWindow("login", "Bejelentkezés")
	p := gwu.NewPanel()
	win.Add(p)
	const placeholder = "sessionid-to-be-changed"
	providers := oidcauth.Providers()
	linkIds := make([]gwu.ID, len(providers))
	for i, provider := range providers {
		authCodeURL := provider.AuthCodeURL(placeholder)
		l := gwu.NewLink(provider.Name, authCodeURL)
		p.Add(l)
		linkIds[i] = l.Id()
	}
	win.AddEHandlerFunc(func(e gwu.Event) {
		sess := e.Session()
		sessID := sess.Id()
		log.Printf("Win LOAD %s", sessID)
		oidcauth.AddSession(sess)

		for _, lID := range linkIds {
			l := p.ById(lID).(gwu.Link)
			u := strings.Replace(l.Url(), placeholder, url.QueryEscape(sessID), 1)
			l.SetUrl(u)
			e.MarkDirty(l)
		}
	},
		gwu.ETypeWinLoad)

	return win
}

func mainWindow() gwu.Window {
	// Create and build a window
	win := gwu.NewWindow("main", "Számlázás")
	win.Style().SetFullWidth()
	win.SetHAlign(gwu.HACenter)
	win.SetCellPadding(2)
	p := gwu.NewPanel()
	win.Add(p)
	l := gwu.NewLabel("Helló!")
	p.Add(l)
	win.AddEHandlerFunc(func(e gwu.Event) {
		sess := e.Session()
		user := oidcauth.GetUser(sess)
		log.Printf("session: %v (%t) token=%#v", sess.Id(), sess.Private(), user)
		l.SetText(fmt.Sprintf("Helló, %s", user.DisplayName()))
		e.MarkDirty(l)
	},
		gwu.ETypeWinLoad)
	return win
}

// A SessionHandler implementation:
type sessHandler struct {
	destURL string
}

func (h sessHandler) Created(s gwu.Session) {
	w := loginWindow(oauth2.NoContext, h.destURL)
	s.AddWin(w)
	s.AddWin(mainWindow())
}

func (h sessHandler) Removed(s gwu.Session) {}
