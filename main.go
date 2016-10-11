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

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/venicegeo/bf-handle/bf"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

func main() {

	catalog.SetImageCatalogPrefix("pzsvc-image-catalog")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// sets up the CORS stuff and stops if it's a Preflighted OPTIONS request
		if pzsvc.Preflight(w, r) {
			return
		}

		w.Header().Set("Content-Type", "application/json")

		r.ParseForm()
		pathStrs := strings.Split(r.URL.Path, "/")
		if len(pathStrs) < 2 {
			fmt.Fprintf(w, `{"greetings":"hello."}`)
			return
		}

		switch pathStrs[1] {
		case "execute":
			bf.Execute(w, r)
		case "executeAsynch":
			bf.HandleAsynch(w, r)
		case "executeBatch":
			bf.ExecuteBatch(w, r)
		case "prepareFootprints":
			bf.PrepareFootprints(w, r)
		case "assembleShorelines":
			bf.AssembleShorelines(w, r)
		case "resultsByScene":
			bf.ResultsByScene(w, r)

		default:
			pzsvc.HTTPOut(w, `{"Errors": "Command undefined.  Try help?",  "Given Path":"`+r.URL.Path+`"}`, http.StatusBadRequest)
		}
	})

	portStr := ":8085"
	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		portStr = fmt.Sprintf(":%s", portEnv)
	}

	log.Fatal(http.ListenAndServe(portStr, nil))
}
