package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

var errInvalidIDParam = errors.New("invalid id path parameter")

func parseIDParam(r *http.Request, name string) (int64, error) {
	value := mux.Vars(r)[name]
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errInvalidIDParam
	}

	return id, nil
}

func parseAccountID(r *http.Request) (int64, error) {
	return parseIDParam(r, "accountId")
}

func parseCardID(r *http.Request) (int64, error) {
	return parseIDParam(r, "cardId")
}

func parseCreditID(r *http.Request) (int64, error) {
	return parseIDParam(r, "creditId")
}
