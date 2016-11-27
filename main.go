package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	oidc "github.com/coreos/go-oidc"
	"github.com/icza/gowut/gwu"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var clientID, clientSecret string

func main() {
	flagTLSCert := flag.String("tls-cert", "", "TLS certificate file")
	flagTLSKey := flag.String("tls-key", "", "TLS key file")
	flag.StringVar(&clientID, "client-id", os.Getenv("CLIENT_ID"), "client ID")
	flag.StringVar(&clientSecret, "client-secret", os.Getenv("CLIENT_SECRET"), "client Secret")
	flag.Parse()

	addr := flag.Arg(0)
	if addr == "" {
		addr = "localhost:8081"
	}
	if err := Main(addr, *flagTLSCert, *flagTLSKey); err != nil {
		log.Fatalf("%+v", err)
	}
}

var providers = make(map[string]*oidc.Provider)

func init() {
	var err error
	providers["google"], err = oidc.NewProvider(context.Background(),
		"https://accounts.google.com")
	if err != nil {
		panic(errors.Wrap(err, "google"))
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
	ch := make(chan oidcToken, 1)
	server.AddSessCreatorName("login", "Login Window")
	server.AddSHandler(sessHandler{
		baseURL: baseURL, destURL: server.AppUrl(), ch: ch,
	})
	server.SetLogger(log.New(os.Stderr, "[szamlazo] ", log.LstdFlags))

	server.AddWin(mainWindow())
	return server.Start("")
}

func loginWindow(ctx context.Context, ch chan<- oidcToken, destURL, baseURL string) gwu.Window {
	if clientID == "" || clientSecret == "" {
		log.Println("Empty Client ID/Secret means no auth will work!")
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}

	state := base64.URLEncoding.EncodeToString(b[:])

	win := gwu.NewWindow("login", "Bejelentkezés")
	p := gwu.NewPanel()
	win.Add(p)

	URL, err := url.Parse(baseURL)
	if err != nil {
		panic(fmt.Sprintf("parse %q: %v", baseURL, err))
	}
	for name, provider := range providers {
		redirectURL := baseURL + "/_auth/" + name + "/callback"
		conf := oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  redirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		}
		authCodeURL := conf.AuthCodeURL(state)
		cbURL := URL.Path + "/_auth/" + name + "/callback"
		log.Println(cbURL)
		http.Handle(cbURL,
			authCallback(ctx, ch, conf, destURL, state,
				provider.Verifier(
					oidc.VerifyExpiry(),
					oidc.VerifySigningAlg(
						oidc.RS256, oidc.RS384, oidc.RS512,
						oidc.ES256, oidc.ES384, oidc.ES512,
						oidc.PS256, oidc.PS384, oidc.PS512,
					),
				),
			),
		)

		log.Printf("adding %q provider", name)
		p.Add(gwu.NewLink(name, authCodeURL))
	}
	return win
}

func authCallback(ctx context.Context, ch chan<- oidcToken, config oauth2.Config, destURL, state string, verifier *oidc.IDTokenVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := oidcToken{Token: oauth2Token, IDTokenClaims: new(json.RawMessage)}
		if err := idToken.Claims(&resp.IDTokenClaims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ch <- resp

		data, _ := json.Marshal(resp)
		log.Println(string(data))

		if destURL != "" {
			http.Redirect(w, r, destURL, http.StatusFound)
			return
		}
		// Just for printing out
		oauth2Token.AccessToken = "*REDACTED*"
		data, err = json.MarshalIndent(resp, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})
}

type oidcToken struct {
	*oauth2.Token
	IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
}

func mainWindow() gwu.Window {
	// Create and build a window
	win := gwu.NewWindow("main", "Számlázás")
	win.Style().SetFullWidth()
	win.SetHAlign(gwu.HACenter)
	win.SetCellPadding(2)
	p := gwu.NewPanel()
	win.Add(p)
	l := gwu.NewLabel("token")
	p.Add(l)
	l.AddEHandlerFunc(func(e gwu.Event) {
		log.Printf("session: %v (%t)", e.Session().Id(), e.Session().Private())
		l.SetToolTip(fmt.Sprintf("%v", e.Session().Attr("oidc_token")))
	},
		gwu.ETypeMouseOver)
	return win
}

// A SessionHandler implementation:
type sessHandler struct {
	baseURL, destURL string
	ch               chan oidcToken
}

func (h sessHandler) Created(s gwu.Session) {
	log.Printf("Session %q created.", s.Id())
	s.AddWin(loginWindow(oauth2.NoContext, h.ch, h.destURL, h.baseURL))
	go func() {
		tok := <-h.ch
		log.Printf("Set oidc_token on %v: %#v", s.Id(), tok)
		s.SetAttr("oidc_token", tok)
	}()
}

func (h sessHandler) Removed(s gwu.Session) {}
