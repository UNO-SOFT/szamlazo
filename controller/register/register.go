// Package register handles the user creation.
package register

import (
	"errors"
	"net/http"

	"github.com/UNO-SOFT/szamlazo/lib/flight"
	"github.com/UNO-SOFT/szamlazo/middleware/acl"
	"github.com/UNO-SOFT/szamlazo/model"

	"github.com/blue-jay/core/form"
	"github.com/blue-jay/core/passhash"
	"github.com/blue-jay/core/router"
)

// Load the routes.
func Load() {
	router.Get("/register", Index, acl.DisallowAuth)
	router.Post("/register", Store, acl.DisallowAuth)
}

// Index displays the register page.
func Index(w http.ResponseWriter, r *http.Request) {
	c := flight.Context(w, r)
	v := c.View.New("register/index")
	form.Repopulate(r.Form, v.Vars, "first_name", "last_name", "email")
	v.Render(w, r)
}

// Store handles the registration form submission.
func Store(w http.ResponseWriter, r *http.Request) {
	c := flight.Context(w, r)

	// Validate with required fields
	if valid, missingField := form.Required(r, "first_name", "last_name", "email", "password", "password_verify"); !valid {
		c.FlashError(errors.New("Field missing: " + missingField))
		Index(w, r)
		return
	}

	// Get form values
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	email := r.FormValue("email")

	// Validate passwords
	if r.FormValue("password") != r.FormValue("password_verify") {
		c.FlashError(errors.New("Passwords do not match."))
		Index(w, r)
		return
	}

	// Hash password
	password, errp := passhash.HashString(r.FormValue("password"))

	// If password hashing failed
	if errp != nil {
		c.FlashError(errp)
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	// Get database result
	_, noRows, err := model.User.ByEmail(email)

	if noRows { // If success (no user exists with that email)
		_, err = model.User.Create(firstName, lastName, email, password)
		// Will only error if there is a problem with the query
		if err != nil {
			c.FlashError(err)
		} else {
			c.FlashSuccess("Account created successfully for: " + email)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	} else if err != nil { // Catch all other errors
		c.FlashError(err)
	} else { // Else the user already exists
		c.FlashError(errors.New("Account already exists for: " + email))
	}

	// Display the page
	Index(w, r)
}
