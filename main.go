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
	//	"encoding/json"
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
			/*		case "/newProductLine":
						bf.NewProductLine(w, r)
					case "/getProductLines":
						bf.GetProductLines(w, r)
					case "/eventTypes":
						pzsvc.WriteEventTypes(w, r)*/
		case "resultsByScene":
			bf.ResultsByScene(w, r)
			/*		case "/resultsByProductLine":
					// extract trigger Id, number per page, and page length
					// search alerts by trigger Id, order by createdOn, demarshal to list of appropriate objects
					// build list of jobIDs
					// return appropriate object
					type PljStruct struct {
						TriggerID   string
						PerPage     string
						PageNo      string
						PzAddr      string
						PzAuthToken string
					}
					var (
						inpObj  PljStruct
						outData = make([]string, 0)
					)

					if b, err := pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
						pzsvc.HTTPOut(w, `{"Errors": "pzsvc.ReadBodyJSON: `+err.Error()+`.",  "Input String":"`+string(b)+`"}`, http.StatusBadRequest)
						return
					}

					alertList, err := pzsvc.GetAlerts(inpObj.PerPage, inpObj.PageNo, inpObj.TriggerID, inpObj.PzAddr, inpObj.PzAuthToken)
					if err != nil {
						pzsvc.HTTPOut(w, `{"Errors": "pzsvc.GetAlerts: `+err.Error()+`"}`, http.StatusBadRequest)
						return
					}

					for _, alert := range alertList {
						var outpObj struct {
							Data pzsvc.JobStatusResp `json:"data,omitempty"`
						}
						_, err = pzsvc.RequestKnownJSON("GET", "", inpObj.PzAddr+"/job/"+alert.JobID, inpObj.PzAuthToken, &outpObj)
						if err != nil {
							continue
						}
						if outpObj.Data.Status == "Success" && outpObj.Data.Result != nil {
							outData = append(outData, outpObj.Data.Result.DataID)
						}
					}
					byts, err := json.Marshal(outData)
					if err != nil {
						pzsvc.HTTPOut(w, `{"Errors": "json.Marshal: `+err.Error()+`.",  "Input String":"`+string(byts)+`"}`, http.StatusBadRequest)
						return
					}

					pzsvc.HTTPOut(w, string(byts), http.StatusOK)*/

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
