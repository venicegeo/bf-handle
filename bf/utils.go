// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bf

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/venicegeo/pzsvc-lib"
)

// handleOut is a function for making sure that output is
// handled in a consistent manner.
func handleOut(w http.ResponseWriter, errmsg string, outpObj interface{}, status int) {
	b, err := json.Marshal(outpObj)
	var outStr string

	if err != nil {
		outStr = `{"error":"json.Marshal error: ` + jsonEscString(err.Error()) + `", "baseError":"` + jsonEscString(errmsg) + `"}`
	} else {
		// Rather than trying to manage any sort of pretense at polymorphism in Go,
		// we just slice off the starter open-brace, and slap the error in manually.
		outStr = `{"error":"` + jsonEscString(errmsg) + `",` + string(b[1:])
	}

	pzsvc.HTTPOut(w, outStr, status)
	return
}

func jsonEscString(modString string) string {
	modString = strings.Replace(modString, `\`, `\\`, -1)
	modString = strings.Replace(modString, `"`, `\"`, -1)
	return modString
}