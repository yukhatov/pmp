package models

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"goji.io/pat"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func getUintIDFromRequest(r *http.Request, name string) (uint64, error) {
	id := pat.Param(r, name)

	if id == "" {
		return 0, errors.New(fmt.Sprintf("field %s is required", name))
	}
	return strconv.ParseUint(id, 10, 0)
}

func getFloatValueFromForm(r *http.Request, name string, isRequired bool) (float64, error) {
	value := r.Form.Get(name)
	if isRequired && value == "" {
		return 0.0, errors.New(fmt.Sprintf("field %s is required", name))
	}
	return strconv.ParseFloat(value, 64)
}

func getUintValueFromForm(r *http.Request, name string, isRequired bool) (uint64, error) {
	value := r.Form.Get(name)
	if isRequired && value == "" {
		return 0.0, errors.New(fmt.Sprintf("field %s is required", name))
	}
	return strconv.ParseUint(value, 10, 0)
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
