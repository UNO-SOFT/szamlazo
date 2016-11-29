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
	"strings"
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

var providers = make(map[string]Provider)

type Provider struct {
	*oidc.Provider
	*oauth2.Config
}

func (p Provider) Endpoint() oauth2.Endpoint {
	return p.Provider.Endpoint()
}

func init() {
	for name, issuer := range map[string]string{
		"Google":     "https://accounts.google.com",
		"eBay":       "https://openidconnect.ebay.com",
		"SalesForce": "https://login.salesforce.com",
		"Microsoft":  "https://sts.windows.net/" + msTenantID,
	} {
		prov, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			log.Printf("%s: %v", errors.Wrap(err, issuer))
			continue
		}
		if prov == nil {
			continue
		}
		providers[name] = Provider{Provider: prov}
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
	registerAuthCbHandlers(context.Background(), http.DefaultServeMux, sH.destURL, baseURL)

	return server.Start("")
}

func registerAuthCbHandlers(ctx context.Context, mux *http.ServeMux, destURL, baseURL string) {
	URL, err := url.Parse(baseURL)
	if err != nil {
		panic(fmt.Sprintf("parse %q: %v", baseURL, err))
	}
	for name, provider := range providers {
		redirectURL := baseURL + "/_auth/" + strings.ToLower(name) + "/callback"
		conf := provider.Config
		if conf == nil {
			conf = &oauth2.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Endpoint:     provider.Endpoint(),
				RedirectURL:  redirectURL,
				Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			}
			provider.Config = conf
			providers[name] = provider
		}
		cbURL := URL.Path + "/_auth/" + name + "/callback"
		log.Printf("adding %q provider as %q", name, cbURL)
		provider := provider
		http.Handle(cbURL,
			authCallback(ctx, *conf, destURL,
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
	p := gwu.NewPanel()
	win.Add(p)
	const placeholder = "sessionid-to-be-changed"
	linkIds := make([]gwu.ID, 0, len(providers))
	for name, provider := range providers {
		authCodeURL := provider.Config.AuthCodeURL(placeholder)
		l := gwu.NewLink(name, authCodeURL)
		p.Add(l)
		linkIds = append(linkIds, l.Id())
	}
	win.AddEHandlerFunc(func(e gwu.Event) {
		log.Printf("Win LOAD %s", e.Session().Id())
		sessionCodeMu.Lock()
		sessionCode[state] = e.Session()
		sessionCodeMu.Unlock()

		for _, lID := range linkIds {
			l := p.ById(lID).(gwu.Link)
			u := strings.Replace(l.Url(), placeholder, url.QueryEscape(e.Session().Id()), 1)
			l.SetUrl(u)
			e.MarkDirty(l)
		}
	},
		gwu.ETypeWinLoad)

	return win
}

func authCallback(ctx context.Context, config oauth2.Config, destURL string, verifier *oidc.IDTokenVerifier) http.Handler {
	log.Printf("authCallback destURL=%q", destURL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionid := r.URL.Query().Get("state")
		sessionCodeMu.Lock()
		sess, ok := sessionCode[sessionid]
		if ok {
			delete(sessionCode, sessionid)
		}
		sessionCodeMu.Unlock()
		if !ok {
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
		log.Printf("set %s.oidc_token = %#v", sess.Id(), resp)
		sess.SetAttr("user", resp.UserInfo)
		sess.RemoveWin(sess.WinByName("login"))

		data, _ := json.Marshal(resp)
		log.Println(string(data))

		if destURL != "" {
			log.Printf("Redirecting to %q", destURL)
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
		user, _ := sess.Attr("user").(UserInfo)
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
