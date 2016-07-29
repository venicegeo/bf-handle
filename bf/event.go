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
	Title		string		`json:"title,omitempty"`
}

// TODO: currently, this is at best half-built.  It won't (and can't) be
// finalized until we have an actual format for the incoming events, and
// have figured out a format for the trigReqStruct object.
func buildTriggerRequestJSON (trigData trigReqStruct, layerGroupID string) string {

	queryFilters := []string(nil)
	if trigData.SensorName != "" {
		qString := `{"match":{"SensorName":"` + trigData.SensorName + `"}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.CloudCover != "" {
		qString := `{"range":{"cloudCover":{"lte":` + trigData.CloudCover + `}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxX != "" {
		qString := `{"range":{"MinX":{"lte":` + trigData.MaxX + `}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinX != "" {
		qString := `{"range":{"MaxX":{"gte":` + trigData.MinX + `}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MaxY != "" {
		qString := `{"range":{"MinY":{"lte":` + trigData.MaxY + `}}}`
		queryFilters = append(queryFilters, qString)
	}
	if trigData.MinY != "" {
		qString := `{"range":{"MaxY":{"gte":` + trigData.MinY + `}}}`
		queryFilters = append(queryFilters, qString)
	}

	resCheck := []string(nil)
	if trigData.MaxRes != "" {
		resCheck = append(resCheck, `"lte":"` + trigData.MaxRes + `"`)
	}
	if trigData.MinRes != "" {
		resCheck = append(resCheck, `"gte":"` + trigData.MinRes + `"`)
	}
	if resCheck != nil {
		qString := `{"range":{"resolution":{` + pzsvc.SliceToCommaSep(resCheck) + `}}}`
		queryFilters = append(queryFilters, qString)
	}

	timeCheck := ""
	if trigData.MaxRes != "" {
		timeCheck = timeCheck + `"lte":"` + trigData.MaxDate + `",`
	}
	if trigData.MinRes != "" {
		timeCheck = timeCheck + `"gte":"` + trigData.MinDate + `",`
	}
	if timeCheck != "" {
		qString := `{"range":{"acquiredDate":{` + timeCheck + `"format":"yyyy-MM-dd'T'HH:mm:ssZZ"}}}`
		queryFilters = append(queryFilters, qString)
	}

	qString := `"query":{"query":{"bool":{"filter":[` + pzsvc.SliceToCommaSep(queryFilters) + `]}}}`
	condString := `"condition":{"eventTypeIds":["` + trigData.EventTypeID + `"],` + qString + `},`

	bfInpObj := &trigData.BFinpObj
	bfInpObj.LGroupID = layerGroupID
	bfInpObj.MetaURL = "$link"

	b, _ := json.Marshal(bfInpObj)

	type datInpType struct{ Content string `json:"content"`; Type string `json:"type"`; MimeType string `json:"mimeType"` }

	datInpObj := datInpType{ string(b), "text", "application/json" }
	
	b2, _ := json.Marshal(datInpObj)

	jobDataInpString := `"dataInputs":{"body":` + string(b2) + `},`
	jobDataOutString := `"dataOutput":[{"mimeType":"application/json","type":"text"}]`
	jobDataString := `"serviceId":"` + trigData.ServiceID + `",` + jobDataInpString + jobDataOutString

	jobString := `"job":{"jobType":{"type":"execute-service","data":{` + jobDataString + `}}}`
	totalString := `{"title":"` + trigData.Title + `","enabled":true,` + condString + jobString + `}`

	return totalString
}

// NewProductLine ....
func NewProductLine (w http.ResponseWriter, r *http.Request) {

	var inpObj trigReqStruct
	type outpType struct {
		TriggerID	string	`json:"triggerId"`
		LayerID		string	`json:"layerId"`
		Error		string	`json:"error"`
	}

	type newTrigData struct {
		ID			string	`json:"triggerId"`
	}
	type newTrigOut struct {
		StatusCode	int			`json:"statusCode"`
		Data		newTrigData	`json:"data"`
	}


	outpObj := outpType{}
	idObj := newTrigOut{}

	// handleOut is a subfunction for making sure that the output is
	// handled in a consistent manner.
	handleOut := func (errmsg string, status int) {
		outpObj.Error = errmsg
		b, err := json.Marshal(outpObj)
fmt.Println(string(b))
		if err != nil {
			b = []byte(`{"error":"json.Marshal error: `+err.Error()+`", "baseError":"`+errmsg+`"}`)
		}
		http.Error(w, string(b), status)
		return
	}

	b2, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
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
fmt.Println(outJSON)

	// TODO: once we can make a few test-runs and get a better idea of the shape of the
	// response object, we may want to do something with them.
	b2, err = pzsvc.RequestKnownJSON("POST", outJSON, bfInpObj.PzAddr + `/trigger`, bfInpObj.PzAuth, &idObj, &http.Client{})
	if err != nil {
		handleOut("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".  http Error: " + string(b2), http.StatusInternalServerError)
		return
	}

	outpObj.TriggerID = idObj.Data.ID
fmt.Println("idObj.ID: " + idObj.Data.ID)

b3, _ := json.Marshal(outpObj)
fmt.Println(string(b3))

	handleOut("", http.StatusOK)
fmt.Println("NewProductLine finished")

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
