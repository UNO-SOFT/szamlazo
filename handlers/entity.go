package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/tbellembois/gowebskel/models"
)

// VGetEntitiesHandler
func (env *Env) VGetEntitiesHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	if e := env.Templates["entities"].ExecuteTemplate(w, "base", nil); e != nil {
		return &models.AppError{
			Error:   e,
			Code:    http.StatusInternalServerError,
			Message: "error executing template base",
		}
	}
	return nil
}

func (env *Env) VCreateEntityHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	if e := env.Templates["entityc"].ExecuteTemplate(w, "base", nil); e != nil {
		return &models.AppError{
			Error:   e,
			Code:    http.StatusInternalServerError,
			Message: "error executing template base",
		}
	}
	return nil
}

// GetEntitiesHandler
func (env *Env) GetEntitiesHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	entities, err := env.DB.GetEntities()
	if err != nil {
		return &models.AppError{
			Error:   err,
			Code:    http.StatusInternalServerError,
			Message: "error getting the entities",
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entities)
	return nil
}

// GetEntityHandler
func (env *Env) GetEntityHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	vars := mux.Vars(r)
	var (
		id  int
		err error
	)

	if id, err = strconv.Atoi(vars["id"]); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "id atoi conversion",
			Code:    http.StatusInternalServerError}
	}

	entity, _ := env.DB.GetEntity(id)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entity)
	return nil
}

// CreateEntityHandler
func (env *Env) CreateEntityHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	var (
		e models.Entity
	)
	if err := r.ParseForm(); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "form parsing error",
			Code:    http.StatusBadRequest}
	}
	var decoder = schema.NewDecoder()
	if err := decoder.Decode(&e, r.PostForm); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "form decoding error",
			Code:    http.StatusBadRequest}
	}

	if err := env.DB.CreateEntity(e); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "create entity error",
			Code:    http.StatusInternalServerError}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(e)
	return nil
}

// UpdateEntityHandler
func (env *Env) UpdateEntityHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	vars := mux.Vars(r)
	var (
		id  int
		err error
		e   models.Entity
	)
	if err := r.ParseForm(); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "form parsing error",
			Code:    http.StatusBadRequest}
	}
	var decoder = schema.NewDecoder()
	if err := decoder.Decode(&e, r.PostForm); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "form decoding error",
			Code:    http.StatusBadRequest}
	}

	if id, err = strconv.Atoi(vars["id"]); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "id atoi conversion",
			Code:    http.StatusInternalServerError}
	}

	updatede, _ := env.DB.GetEntity(id)
	updatede.Name = e.Name
	updatede.Description = e.Description

	if err := env.DB.UpdateEntity(updatede); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "update entity error",
			Code:    http.StatusInternalServerError}
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatede)
	return nil
}

// DeleteEntityHandler
func (env *Env) DeleteEntityHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	vars := mux.Vars(r)
	var (
		id  int
		err error
	)

	if id, err = strconv.Atoi(vars["id"]); err != nil {
		return &models.AppError{
			Error:   err,
			Message: "id atoi conversion",
			Code:    http.StatusInternalServerError}
	}

	env.DB.DeleteEntity(id)
	return nil
}
