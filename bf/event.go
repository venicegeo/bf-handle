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
	"fmt"
	//	"io/ioutil"
	//	"log"
	"net/http"
	"os"
	//	"time"

	"github.com/venicegeo/pzsvc-lib"
	//	"github.com/venicegeo/geojson-go/geojson"
	//	"github.com/venicegeo/pzsvc-image-catalog/catalog"
)

type trigReqStruct struct {
	BFinpObj    gsInpStruct `json:"bfInputJSON,omitempty"`
	MaxX        string      `json:"maxX,omitempty"`
	MinX        string      `json:"minX,omitempty"`
	MaxY        string      `json:"maxY,omitempty"`
	MinY        string      `json:"minY,omitempty"`
	CloudCover  string      `json:"cloudCover,omitempty"`
	MaxRes      string      `json:"maxRes,omitempty"`
	MinRes      string      `json:"minRes,omitempty"`
	MaxDate     string      `json:"maxDate,omitempty"`
	MinDate     string      `json:"mainDate,omitempty"`
	SensorName  string      `json:"sensorName,omitempty"`
	EventTypeID string      `json:"eventTypeId,omitempty"`
	ServiceID   string      `json:"serviceId,omitempty"`
	Name        string      `json:"name,omitempty"`
}

/*
- Format for the feeding EventType?
--- "imageID":"string"
--- "acquiredDate":"string"
--- "cloudCover":"long"
--- "resolution":"long"
--- "sensorName":"string"
--- "minx":"long"
--- "miny":"long"
--- "maxx":"long"
--- "maxy":"long"
--- "link":"string"
*/

func buildTriggerRequestJSON(trigData trigReqStruct, layerGroupID string) string {

	var trigObj pzsvc.Trigger
	trigObj.Name = trigData.Name
	trigObj.Enabled = true
	trigObj.Condition.EventTypeIDs = []string{trigData.EventTypeID}

	queryFilters := []pzsvc.QueryClause{}
	if trigData.SensorName != "" {
		sensorMatch := map[string]string{"SensorName": trigData.SensorName}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: sensorMatch, Range: nil})
	}
	if trigData.CloudCover != "" {
		cClause := pzsvc.CompClause{LTE: trigData.CloudCover, GTE: nil, Format: ""}
		cloudRange := map[string]pzsvc.CompClause{"cloudCover": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: cloudRange})
	}
	if trigData.MaxX != "" {
		cClause := pzsvc.CompClause{LTE: trigData.MaxX, GTE: nil, Format: ""}
		XRange := map[string]pzsvc.CompClause{"MinX": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: XRange})
	}
	if trigData.MinX != "" {
		cClause := pzsvc.CompClause{LTE: nil, GTE: trigData.MinX, Format: ""}
		XRange := map[string]pzsvc.CompClause{"MaxX": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: XRange})
	}
	if trigData.MaxY != "" {
		cClause := pzsvc.CompClause{LTE: trigData.MaxY, GTE: nil, Format: ""}
		YRange := map[string]pzsvc.CompClause{"MinY": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: YRange})
	}
	if trigData.MinY != "" {
		cClause := pzsvc.CompClause{LTE: nil, GTE: trigData.MinY, Format: ""}
		YRange := map[string]pzsvc.CompClause{"MaxY": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: YRange})
	}

	if trigData.MaxRes != "" || trigData.MinRes != "" {
		resClause := pzsvc.CompClause{LTE: nil, GTE: nil, Format: ""}
		if trigData.MaxRes != "" {
			resClause.LTE = trigData.MaxRes
		}
		if trigData.MinRes != "" {
			resClause.GTE = trigData.MinRes
		}
		resFilter := map[string]pzsvc.CompClause{"resolution": resClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: resFilter})
	}

	if trigData.MaxDate != "" || trigData.MinDate != "" {
		dateClause := pzsvc.CompClause{LTE: nil, GTE: nil, Format: "yyyy-MM-dd'T'HH:mm:ssZZ"}
		if trigData.MaxDate != "" {
			dateClause.LTE = trigData.MaxDate
		}
		if trigData.MinDate != "" {
			dateClause.GTE = trigData.MinDate
		}
		dateFilter := map[string]pzsvc.CompClause{"acquiredDate": dateClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: dateFilter})
	}

	trigObj.Condition.Query.Query.Bool.Filter = queryFilters

	trigObj.Job.JobType.Type = "execute-service"

	bfInpObj := &trigData.BFinpObj
	bfInpObj.LGroupID = layerGroupID
	bfInpObj.MetaURL = "$link"
	b, _ := json.Marshal(bfInpObj)

	jobInpObj := pzsvc.DataType{Content: string(b), Type: "text", MimeType: "application/json"}
	jobOutpObj := pzsvc.DataType{Content: "", Type: "text", MimeType: "application/json"}
	jobIntMap := map[string]pzsvc.DataType{"body": jobInpObj}
	trigObj.Job.JobType.Data = pzsvc.JobData{ServiceID: trigData.ServiceID, DataInputs: jobIntMap, DataOutput: []pzsvc.DataType{jobOutpObj}}

	b2, _ := json.Marshal(trigObj)
	return string(b2)
}

// NewProductLine ....
func NewProductLine(w http.ResponseWriter, r *http.Request) {

	type outpType struct {
		TriggerID string `json:"triggerId"`
		LayerID   string `json:"layerId"`
	}

	type newTrigData struct {
		ID string `json:"triggerId"`
	}
	type newTrigOut struct {
		StatusCode int         `json:"statusCode"`
		Data       newTrigData `json:"data"`
	}

	inpObj := trigReqStruct{}
	outpObj := outpType{}
	idObj := newTrigOut{}

	_, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}

	bfInpObj := &inpObj.BFinpObj

	if bfInpObj.PzAuth == "" {
		bfInpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if bfInpObj.DbAuth == "" {
		bfInpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	layerID, err := pzsvc.AddGeoServerLayerGroup(bfInpObj.PzAddr, bfInpObj.PzAuth)
	if err != nil {
		handleOut(w, "Error: pzsvc.AddGeoServerLayerGroup: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}

	outJSON := buildTriggerRequestJSON(inpObj, layerID)
	fmt.Println(outJSON)

	// TODO: once we can make a few test-runs and get a better idea of the shape of the
	// response object, we may want to do something with them.
	b, err := pzsvc.RequestKnownJSON("POST", outJSON, bfInpObj.PzAddr+`/trigger`, bfInpObj.PzAuth, &idObj)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error()+".  http Error: "+string(b), outpObj, http.StatusInternalServerError)
		return
	}

	outpObj.TriggerID = idObj.Data.ID
	fmt.Println("idObj.ID: " + idObj.Data.ID)

	b3, _ := json.Marshal(outpObj)
	fmt.Println(string(b3))

	handleOut(w, "", outpObj, http.StatusOK)
	fmt.Println("NewProductLine finished")

}

// GetTriggers responds to a properly formed network request
// by sending out a list of triggers in JSON format.
func GetTriggers(w http.ResponseWriter, r *http.Request) {

	var inpObj struct {
		EventTypeID string `json:"eventTypeId"`
		ServiceID   string `json:"serviceId"`
		CreatedBy   string `json:"createdBy"`
		PageNo      int    `json:"pageNo"`
		PerPage     int    `json:"perPage"`
		PzAddr      string `json:"pzAddr"`
		PzAuth      string `json:"pzAuth"`
		Order       string `json:"order"`
		SortBy      string `json:"sortBy"`
	}

	var outpObj struct {
		TrigList []trigReqStruct `json:"triggerList"`
	}

	_, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	// set up output obj.
	// set up input obj.

	/*
		b, err := pzsvc.RequestKnownJSON("POST", outJSON, bfInpObj.PzAddr + `/trigger`, bfInpObj.PzAuth, &idObj, &http.Client{})
		if err != nil {
			handleOut(w, "Error: pzsvc.ReadBodyJSON: " + err.Error() + ".  http Error: " + string(b), outpObj, http.StatusInternalServerError)
			return
		}
	*/

	// request all triggers by EventTypeID/pageNo/perPage
	// demarshal and break down into list of trigger objects
	// large for range triggerList loop
	// - filter out any triggers that don't belong based on searched list
	// --- keep them simple.  EventTypeId, ServiceId, CreatedBy.  Should be plenty to start.
	// - break it back down into a trigReqStruct, add to list of trigReqStruct
	// marshal list of trigReqStruct, and send as response
	// return
}

// handleOut is a function for making sure that output is
// handled in a consistent manner.
func handleOut(w http.ResponseWriter, errmsg string, outpObj interface{}, status int) {
	b, err := json.Marshal(outpObj)
	var outStr string

	if err != nil {
		outStr = `{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`
	} else {
		// Rather than trying to manage any sort fo pretense at polymorphism in Go,
		// we just slice off the starter open-brace, and slap the error in manually.
		outStr = `{"error":"` + errmsg + `",` + string(b[1:])
	}

	http.Error(w, outStr, status)
	return
}
