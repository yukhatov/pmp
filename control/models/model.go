package models

import "net/http"

type Model interface {
	PopulateData(r *http.Request)
	UpdateData(r *http.Request)
	GetByID(id interface{})
	Save()
	Create()
}
