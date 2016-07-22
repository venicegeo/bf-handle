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

	BFinpObj	gsInpStruct	`json:"bfInputJSON,omitempty"`	
	MaxX		string		`json:"maxX,omitempty"`
	MinX		string		`json:"minX,omitempty"`
	MaxY		string		`json:"maxY,omitempty"`
	MinY		string		`json:"minY,omitempty"`
	CloudCover	string		`json:"cloudCover,omitempty"`
	MaxRes		string		`json:"maxRes,omitempty"`
	MinRes		string		`json:"minRes,omitempty"`
	MaxDate		string		`json:"maxDate,omitempty"`
	MinDate		string		`json:"mainDate,omitempty"`
	SensorName	string		`json:"sensorName,omitempty"`
	EventTypeID	string		`json:"eventTypeId,omitempty"`
	ServiceID	string		`json:"serviceId,omitempty"`
}

// TODO: currently, this is at best half-built.  It won't (and can't) be
// finalized until we have an actual format for the incoming events, and
// have figured out a format for the trigReqStruct object.
func buildTriggerRequestJSON (trigData trigReqStruct, layerGroupID string) string {

	queryFilters := make([]string, 10)
	if trigData.SensorName != "" {
		qString := `{ "match": { "SensorName" : "` + trigData.SensorName + `" } }`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.CloudCover != "" {
		qString := `{"range": {"cloudCover": {"lte":"` + trigData.CloudCover + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxX != "" {
		qString := `{"range": {"MinX": {"lte":"` + trigData.MaxX + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinX != "" {
		qString := `{"range": {"MaxX": {"gte":"` + trigData.MinX + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxY != "" {
		qString := `{"range": {"MinY": {"lte":"` + trigData.MaxY + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinY != "" {
		qString := `{"range": {"MaxY": {"gte":"` + trigData.MinY + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxRes != "" {
		qString := `{"range": {"resolution": {"lte":"` + trigData.MaxRes + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinRes != "" {
		qString := `{"range": {"resolution": {"gte":"` + trigData.MinRes + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxDate != "" {
		qString := `{"range": {"acquiredDate": {"lte":"` + trigData.MaxDate + `"}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinDate != "" {
		qString := `{"range": {"acquiredDate": {"gte":"` + trigData.MinDate + `"}}}`
		queryFilters = append(queryFilters, qString)
	}

	qString := `"query": { "query": { "bool": { "filter": [` + pzsvc.SliceToCommaSep(queryFilters) + `] } } }`
	condString := `"condition": { "eventtype_ids": ["` + trigData.EventTypeID + `"], ` + qString + ` }, `

	bfInpObj := &trigData.BFinpObj
	bfInpObj.LGroupID = layerGroupID

	// TODO: event only gives link to place to get geojson feature.  Will need to modify GenShoreline accordingly
	//- figure out what you need to do to get a variable in place as the URL
	//- put the event variable in place as MetaURL

	b, _ := json.Marshal(bfInpObj)
	datInpObj := struct{ Content string `json:"content"`; Type string `json:"type"` }{ string(b), "urlparameter" }
	b2, _ := json.Marshal(datInpObj)
	jobDataInpString := `"dataInputs": {"body": ` + string(b2) + ` },`
	jobDataOutString := `"dataOutput": [ { "mimeType": "application/json", "type": "text" } ]`
	jobDataString := `"serviceId": "` + trigData.ServiceID + `", ` + jobDataInpString + jobDataOutString

	jobString := `"job":{ "userName": "", "jobType": { "type": "execute-service", "data": {` + jobDataString + `} } }`
	totalString := `{"id":"", "title": "", ` + condString + jobString + `}`

	return totalString
}

// NewProductLine ....
func NewProductLine (w http.ResponseWriter, r *http.Request) {

	var inpObj trigReqStruct
	var outpObj struct {
		TriggerID	string	`json:"triggerId,omitempty"`
		LayerID		string	`json:"triggerId,omitempty"`
		Error		string	`json:"triggerId,omitempty"`
	}
	var idObj struct {
		ID			string	`json:"id,omitempty"`
	}

	// handleOut is a subfunction for making sure that the output is
	// handled in a consistent manner.
	handleOut := func (errmsg string, status int) {
		outpObj.Error = errmsg
		b, err := json.Marshal(outpObj)
		if err != nil {
			fmt.Fprintf(w, `{"error":"json.Marshal error: `+err.Error()+`", "baseError":"`+errmsg+`"}`)
		}
		http.Error(w, string(b), status)
		return
	}

	_, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut("Error: pzsvc.ReadBodyJSON: " + err.Error(), http.StatusBadRequest)
		return
	}

	bfInpObj := &inpObj.BFinpObj

	if bfInpObj.PzAuth == "" {
		bfInpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if bfInpObj.DbAuth == "" {
		bfInpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	layerID, err := pzsvc.AddGeoServerLayerGroup(bfInpObj.PzAddr, bfInpObj.PzAuth, &http.Client{})
	if err != nil {
		handleOut("Error: pzsvc.AddGeoServerLayerGroup: " + err.Error(), http.StatusBadRequest)
		return
	}

	outJSON := buildTriggerRequestJSON(inpObj, layerID)

	// TODO: once we can make a few test-runs and get a better idea of the shape of the
	// response object, we may want to do something with them.
	_, err = pzsvc.RequestKnownJSON("POST", outJSON, bfInpObj.PzAddr + `/trigger`, bfInpObj.PzAuth, &idObj, &http.Client{})
	if err != nil {
		handleOut("Error: pzsvc.ReadBodyJSON: " + err.Error(), http.StatusInternalServerError)
		return
	}

	outpObj.TriggerID = idObj.ID

}

/*

Things needed:
- knowing the format for the data inputs (defined by the "create trigger" API).
--- BLOCKED: waiting on official Pz documentation
- knowing the format for the incoming data (defined by the Event and Event Type that Jeff's putting together).
--- BLOCKED: Jeff still having trouble actually putting it together
- buildTrigFuncData: specifying where to find the image catalog data
--- BLOCKED: on both of the previous two
- NewProductLine: creating it, sending it where it needs to go, returning it
--- BLOCKED: no "establish Geoserver layer" call to make.
- buildTriggerRequestJSON: fitting it into the request Json properly
--- BLOCKED (mostly): not entirely clear ont eh input formats, dont' have the inputs on the other end,
	messy to do without the NewProductLine part done first.
- bf-handle main and GenShoreline: set up uploads to Geoserver, assigning to appropriate layers
--- BLOCKED: no way to assign to specific layers yet.  
- Once we have it all more or less put together, look at cleaning up the input struct.



Pertinent Questions:
- PercolationID - what is it?
--- it's the ID of the condition in the condition list.  Each time new trigger is added, condition is
	added to condition list. Each time event fires, is compared to each condition in list.  If gets a
	bit, uses PercolationID to find trigger.

- Are we manually specifying CreatedBy?  Seems like that would be automated, based on whose auth was used
--- ???

- What is the format for the feeding EventType?
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
