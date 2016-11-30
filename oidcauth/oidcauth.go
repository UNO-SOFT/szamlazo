package oidcauth

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"

	oidc "github.com/coreos/go-oidc"
	"github.com/icza/gowut/gwu"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

var (
	providers   = make([]Provider, 0, 8)
	providersMu sync.RWMutex

	sessionCodeMu sync.Mutex
	sessionCode   = make(map[string]gwu.Session)
)

type Provider struct {
	Name                   string
	ClientID, ClientSecret string
	*oidc.Provider
	*oauth2.Config
}

func (p Provider) Endpoint() oauth2.Endpoint {
	return p.Provider.Endpoint()
}

// Providers returns the registered Providers.
func Providers() []Provider {
	return providers
}

// Register a new provider.
func RegisterProvider(ctx context.Context, name, issuer, clientID, clientSecret string, scopes ...string) error {
	if issuer == "" {
		issuer = Issuers[name]
		if issuer == "" {
			return errors.Errorf("No issuer specified, and %q is unknown!", name)
		}
	}
	providersMu.Lock()
	defer providersMu.Unlock()
	for _, p := range providers {
		if p.Name == name {
			return errors.Errorf("Provider %q already exist!", name)
		}
	}
	prov, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return errors.Wrap(err, name)
	}
	smap := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		smap[s] = struct{}{}
	}
	for _, k := range []string{oidc.ScopeOpenID, "email"} {
		if _, ok := smap[k]; !ok {
			smap[k] = struct{}{}
		}
	}
	scopes = scopes[:0]
	for k := range smap {
		scopes = append(scopes, k)
	}

	providers = append(providers, Provider{
		Name:     name,
		Provider: prov,
		Config: &oauth2.Config{
			ClientID: clientID, ClientSecret: clientSecret,
			Scopes: scopes,
		},
	})
	return nil
}

// Issuers we know about - just for easier registering.
var Issuers = map[string]string{
	"Google":     "https://accounts.google.com",
	"eBay":       "https://openidconnect.ebay.com",
	"SalesForce": "https://login.salesforce.com",
	"Microsoft":  "https://sts.windows.net/{{.TenantID}}",
}

const (
	AuthToken = "AuthenticationToken"
	AuthUser  = "AuthenticatedUser"
)

// RegisterCbHandlers registers the callback handlers for the providers onto the given mux.
// The callback will set the token and the user info on the session,
// at AuthToken and AuthUser keys, and redirect to destURL if not empty.
//
// The callback will be registered as cbURL, "{{.Provider}}" replaced with the provider's name. "lc" function is available in the template to lowercase its argument.
//
// cb is called with the prepared session in the callback.
func RegisterCbHandlers(ctx context.Context, mux *http.ServeMux, cbURL, destURL string, cb func(gwu.Session)) {
	tmpl := template.Must(template.
		New("url").
		Funcs(template.FuncMap{
			"lc": strings.ToLower,
			"uc": strings.ToUpper,
		}).
		Parse(cbURL))
	var buf bytes.Buffer
	providersMu.RLock()
	defer providersMu.RUnlock()
	for i, provider := range providers {
		buf.Reset()
		if err := tmpl.Execute(&buf, struct{ Provider string }{Provider: provider.Name}); err != nil {
			panic(errors.Wrap(err, provider.Name))
		}
		conf := provider.Config
		if conf == nil {
			conf = &oauth2.Config{Scopes: []string{oidc.ScopeOpenID, "email"}}
		}
		redirect := buf.String()
		conf.Endpoint = provider.Endpoint()
		conf.RedirectURL = redirect

		provider.Config = conf
		providers[i] = provider

		path := mustURL(redirect).Path
		log.Printf("adding %q provider with %q callback url, %q as dest url.", provider.Name, redirect, destURL)
		provider := provider
		http.Handle(path,
			authCallback(ctx, *conf, destURL,
				provider.Verifier(
					oidc.VerifyExpiry(),
					oidc.VerifySigningAlg(
						oidc.RS256, oidc.RS384, oidc.RS512,
						oidc.ES256, oidc.ES384, oidc.ES512,
						oidc.PS256, oidc.PS384, oidc.PS512,
					),
				),
				cb,
			),
		)
	}
}

// Add session to be filled by an authentication callback.
func AddSession(sess gwu.Session) {
	sessionCodeMu.Lock()
	sessionCode[sess.Id()] = sess
	sessionCodeMu.Unlock()
}

func authCallback(ctx context.Context, config oauth2.Config, destURL string, verifier *oidc.IDTokenVerifier, cb func(gwu.Session)) http.Handler {
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
			log.Printf("got state=%q, has %q", sessionid, sessionCode)
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
		sess.SetAttr(AuthToken, oauth2Token)

		resp := oidcToken{Token: oauth2Token}
		if err := idToken.Claims(&resp.UserInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("set %s.oidc_token = %#v", sess.Id(), resp)
		sess.SetAttr(AuthUser, resp.UserInfo)
		if cb != nil {
			cb(sess)
		}

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

func GetUser(sess gwu.Session) UserInfo {
	user, _ := sess.Attr(AuthUser).(UserInfo)
	return user
}

func mustURL(s string) *url.URL {
	URL, err := url.Parse(s)
	if err != nil {
		panic(errors.Wrap(err, s))
	}
	return URL
}
