// Copyright External Secrets Inc
// All Rights reserved.
package template

import "encoding/json"

// Verifies if given string is a valid json object.
func isJson(str string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(str), &js) == nil
}
