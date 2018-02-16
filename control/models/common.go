package models

import (
	"encoding/json"
)

type PsqlDbError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
	Detail  string `json:"Detail"`
}

const CODE_UNIQ_VIOLATION = "23505"

func getError(err error) PsqlDbError {
	var result PsqlDbError
	errorJson, _ := json.Marshal(err)

	json.Unmarshal(errorJson, &result)

	if result.Code == CODE_UNIQ_VIOLATION {
		result.Detail = "This custom id value is already taken."
	}

	return result
}
