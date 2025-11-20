// Copyright External Secrets Inc
// All Rights reserved.

// Package template implements the template engine for External Secrets.
package template

import "encoding/json"

// Verifies if given string is a valid json object.
func isJSON(str string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}
