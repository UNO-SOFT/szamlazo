package handlers

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/tbellembois/gowebskel/models"
)

func (env *Env) VTestHandler(w http.ResponseWriter, r *http.Request) *models.AppError {

	log.Info(env.DB.HasPermission("dark.vader@foo.com", "read", "entity", 1))

	return nil
}
