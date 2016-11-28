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
	"sync"

	oidc "github.com/coreos/go-oidc"
	"github.com/icza/gowut/gwu"
	"github.com/pkg/errors"
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
	for name, issuer := range map[string]string{
		"Google":     "https://accounts.google.com",
		"eBay":       "https://openidconnect.ebay.com",
		"SalesForce": "https://login.salesforce.com",
		"Microsoft":  "https://sts.windows.net/" + msTenantID,
	} {
		providers[name], err = oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			log.Printf("%s: %v", errors.Wrap(err, issuer))
			continue
		}
		if providers[name] == nil {
			delete(providers, name)
			continue
		}
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
	server.AddSHandler(sessHandler{
		baseURL: baseURL, destURL: server.AppUrl(),
	})
	server.SetLogger(log.New(os.Stderr, "[szamlazo] ", log.LstdFlags))
	registerAuthCbHandlers(context.Background(), http.DefaultServeMux, server.AppUrl(), baseURL)

	return server.Start("")
}

func registerAuthCbHandlers(ctx context.Context, mux *http.ServeMux, destURL, baseURL string) {
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
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		}
		cbURL := URL.Path + "/_auth/" + name + "/callback"
		log.Printf("adding %q provider as %q", name, cbURL)
		provider := provider
		http.Handle(cbURL,
			authCallback(ctx, conf, destURL,
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
	}
}

var sessionCodeMu sync.Mutex
var sessionCode = make(map[string]gwu.Session)

func loginWindow(ctx context.Context, destURL string) gwu.Window {
	if clientID == "" || clientSecret == "" {
		log.Println("Empty Client ID/Secret means no auth will work!")
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}

	state := base64.URLEncoding.EncodeToString(b[:])

	win := gwu.NewWindow("login", "Bejelentkezés")
	win.AddEHandlerFunc(func(e gwu.Event) {
		log.Printf("Win LOAD %s", e.Session().Id())
		sessionCodeMu.Lock()
		sessionCode[state] = e.Session()
		sessionCodeMu.Unlock()
	},
		gwu.ETypeWinLoad)
	p := gwu.NewPanel()
	win.Add(p)

	for name, provider := range providers {
		authCodeURL := conf.AuthCodeURL("state")
		p.Add(gwu.NewLink(name, authCodeURL))
	}
	return win
}

func authCallback(ctx context.Context, config oauth2.Config, destURL string, verifier *oidc.IDTokenVerifier) http.Handler {
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

		resp := oidcToken{Token: oauth2Token}
		if err := idToken.Claims(&resp.UserInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sessionCodeMu.Lock()
		sess, ok := sessionCode[state]
		if ok {
			delete(sessionCode, state)
		}
		sessionCodeMu.Unlock()
		if sess == nil {
			log.Println("No session for %s", state)
		} else {
			log.Printf("set %s.oidc_token = %#v", sess.Id(), resp)
			sess.SetAttr("user", resp.UserInfo)
		}

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
	UserInfo
}

type UserInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	ID            string `json:"sub"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
}

func (u UserInfo) DisplayName() string {
	if u.Locale == "hu" && u.GivenName != "" && u.FamilyName != "" {
		return u.GivenName + " " + u.FamilyName
	}
	if u.Name == "" {
		return u.Email
	}
	return u.Name
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
		user := sess.Attr("user").(UserInfo)
		log.Printf("session: %v (%t) token=%#v", sess.Id(), sess.Private(), user)
		l.SetText(fmt.Sprintf("Helló, %s", user.DisplayName()))
		e.MarkDirty(l)
	},
		gwu.ETypeWinLoad)
	return win
}

// A SessionHandler implementation:
type sessHandler struct {
	baseURL, destURL string
}

func (h sessHandler) Created(s gwu.Session) {
	log.Printf("Session %q created.", s.Id())
	w := loginWindow(oauth2.NoContext, h.destURL, h.baseURL)
	s.AddWin(w)
	s.AddWin(mainWindow())
}

func (h sessHandler) Removed(s gwu.Session) {}
