package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"
	"github.com/tbellembois/gowebskel/models"
)

var TokenSignKey = []byte("secret")

func (env *Env) GetTokenHandler(w http.ResponseWriter, r *http.Request) *models.AppError {

	// parsing the form
	if e := r.ParseForm(); e != nil {
		return &models.AppError{
			Code:    http.StatusBadRequest,
			Error:   e,
			Message: "error parsing form",
		}
	}

	// decoding the form
	decoder := schema.NewDecoder()
	person := new(models.Person)
	if e := decoder.Decode(person, r.PostForm); e != nil {
		return &models.AppError{
			Code:    http.StatusInternalServerError,
			Error:   e,
			Message: "error decoding form",
		}
	}
	log.WithFields(log.Fields{"person.Email": person.Email}).Debug("GetTokenHandler")

	// authenticating the person

	// create the token
	token := jwt.New(jwt.SigningMethodHS256)

	// create a map to store our claims
	claims := token.Claims.(jwt.MapClaims)

	// set token claims
	claims["email"] = person.Email
	claims["exp"] = time.Now().Add(time.Hour * 8).Unix()

	// sign the token with our secret
	tokenString, _ := token.SignedString(TokenSignKey)

	// finally, write the token to the browser window
	//w.WriteHeader(http.StatusOK)
	//w.Write([]byte(tokenString))
	// finally set the token in a cookie
	// further readings: https://www.calhoun.io/securing-cookies-in-go/
	c := http.Cookie{
		Name:  "token",
		Value: tokenString,
	}
	http.SetCookie(w, &c)

	return nil
}

func (env *Env) AppMiddleware(h models.AppHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if e := h(w, r); e != nil {
			log.Error(e.Message + "-" + e.Error.Error())
			http.Error(w, e.Message, e.Code)
		}
	})
}

func (env *Env) LogingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Debug(req.RequestURI)
		h.ServeHTTP(w, req)
	})
}

func (env *Env) HeadersMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, req)
	})
}

func (env *Env) AuthorizeMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			email       string
			err         error
			reqToken    *http.Cookie
			reqTokenStr string
			token       *jwt.Token
			itemid      int
			perm        string
			permok      bool
		)

		// token regex header version
		//tre := regexp.MustCompile("Bearer [[:alnum:]]\\.[[:alnum:]]\\.[[:alnum:]]")
		// token regex cookie version
		//tre := regexp.MustCompile("token=[[:alnum:]]\\.[[:alnum:]]\\.[[:alnum:]]")
		tre := regexp.MustCompile("token=.+")

		// extracting the token string from Authorization header
		//reqToken := r.Header.Get("Authorization")
		// extracting the token string from cookie
		if reqToken, err = r.Cookie("token"); err != nil {
			http.Error(w, "token not found in cookies, please log in", http.StatusUnauthorized)
			return
		}
		if !tre.MatchString(reqToken.String()) {
			http.Error(w, "token has an invalid format", http.StatusUnauthorized)
			return
		}
		// header version
		//splitToken := strings.Split(reqToken, "Bearer ")
		// cookie version
		splitToken := strings.Split(reqToken.String(), "token=")
		reqTokenStr = splitToken[1]
		token, err = jwt.Parse(reqTokenStr, func(token *jwt.Token) (interface{}, error) {
			return TokenSignKey, nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// getting the claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// then the email claim
			if cemail, ok := claims["email"]; !ok {
				http.Error(w, "email not found in claims", http.StatusBadRequest)
				return
			} else {
				email = cemail.(string)
			}
		} else {
			http.Error(w, "can not extract claims", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(r)
		item := vars["item"]
		view := vars["view"]
		id := vars["id"]
		log.WithFields(log.Fields{"id": id, "item": item, "view": view, "email": email}).Debug("AuthorizeMiddleware")

		if id == "" {
			itemid = -1
		} else {
			if itemid, err = strconv.Atoi(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		switch r.Method {
		case "GET":
			switch view {
			case "v", "":
				perm = "read"
			case "vu":
				perm = "update"
			case "vc":
				perm = "create"
			}
		case "POST":
			perm = "create"
		case "PUT":
			perm = "update"
		case "DELETE":
			perm = "delete"
		default:
			http.Error(w, "unsupported http verb", http.StatusBadRequest)
			return
		}

		if permok, err = env.DB.HasPermission(email, perm, item, itemid); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !permok {
			log.Debug("unauthorized:" + perm + ":" + item + ":" + id)
			http.Error(w, perm+":"+item+":"+id, http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (env *Env) VLoginHandler(w http.ResponseWriter, r *http.Request) *models.AppError {

	if e := env.Templates["login"].ExecuteTemplate(w, "base", nil); e != nil {
		return &models.AppError{
			Error:   e,
			Code:    http.StatusInternalServerError,
			Message: "error executing template base",
		}
	}

	return nil
}
