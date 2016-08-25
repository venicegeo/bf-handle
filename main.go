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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/venicegeo/bf-handle/bf"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		}
		// Stop here if its Preflighted OPTIONS request
		if r.Method == "OPTIONS" {
			return
		}

		w.Header().Set("Content-Type", "application/json")

		r.ParseForm()
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, "hello.")
		case "/execute":
			bf.Execute(w, r)
		case "/executeBatch":
			bf.ExecuteBatch(w, r)
		case "/prepareFootprints":
			bf.PrepareFootprints(w, r)
		case "/assembleShorelines":
			bf.AssembleShorelines(w, r)
		case "/newProductLine":
			fmt.Println("newProduct triggered")
			bf.NewProductLine(w, r)
		case "/getProductLines":
			fmt.Println("product line listing")
			bf.GetProductLines(w, r)
		case "/resultsByImage":
			bf.ResultsByImage(w, r)
		case "/resultsByProductLine":
			// extract trigger Id, number per page, and page length
			// search alerts by trigger Id, order by createdOn, demarshal to list of appropriate objects
			// build list of jobIDs
			// return appropriate object
			type PljStruct struct {
				TriggerID string
				PerPage   string
				PageNo    string
				PzAddr    string
				PzAuth    string
			}
			var inpObj PljStruct

			if b, err := pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
				pzsvc.HTTPOut(w, `{"Errors": "pzsvc.ReadBodyJSON: `+err.Error()+`.",  "Input String":"`+string(b)+`"}`, http.StatusBadRequest)
				return
			}

			alertList, err := pzsvc.GetAlerts(inpObj.PerPage, inpObj.PageNo, inpObj.TriggerID, inpObj.PzAddr, inpObj.PzAuth)
			if err != nil {
				pzsvc.HTTPOut(w, `{"Errors": "pzsvc.GetAlerts: `+err.Error()+`"}`, http.StatusBadRequest)
				return
			}
			outData := []string(nil)
			for _, alert := range alertList {
				var outpObj struct {
					Data pzsvc.JobStatusResp `json:"data,omitempty"`
				}
				_, err := pzsvc.RequestKnownJSON("GET", "", inpObj.PzAddr+"/job/"+alert.JobID, inpObj.PzAuth, &outpObj)
				if err != nil {
					continue
				}
				if outpObj.Data.Status == "Success" && outpObj.Data.Result != nil{
					outData = append(outData, outpObj.Data.Result.DataID)
				}
			}
			b, _ := json.Marshal(outData)

			pzsvc.HTTPOut(w, string(b), http.StatusOK)

		default:
			fmt.Fprintf(w, "Command undefined.  Try help?\n")
		}
	})

	portStr := ":8085"
	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		portStr = fmt.Sprintf(":%s", portEnv)
	}

	catalog.SetImageCatalogPrefix("pzsvc-image-catalog")

	log.Fatal(http.ListenAndServe(portStr, nil))
}
