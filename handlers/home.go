package handlers

import (
	"net/http"

	"github.com/tbellembois/gowebskel/models"
)

// HomeHandler serve the main page
func (env *Env) HomeHandler(w http.ResponseWriter, r *http.Request) *models.AppError {

	if e := env.Templates["home"].ExecuteTemplate(w, "base", nil); e != nil {
		return &models.AppError{
			Error:   e,
			Code:    http.StatusInternalServerError,
			Message: "error executing template base",
		}
	}

	return nil
}
