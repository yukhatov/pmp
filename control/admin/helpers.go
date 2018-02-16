package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"goji.io/pat"
)

func getUintIDFromRequest(r *http.Request, name string) (uint64, error) {
	id := pat.Param(r, name)
	if id == "" {
		return 0, errors.New(fmt.Sprintf("field %s is required", name))
	}
	return strconv.ParseUint(id, 10, 0)
}

func getStringFromRequest(r *http.Request, name string) (string, error) {
	value := pat.Param(r, name)
	if value == "" {
		return "", errors.New(fmt.Sprintf("field %s is required", name))
	}
	return value, nil
}
