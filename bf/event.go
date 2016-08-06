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
	Name		string		`json:"name,omitempty"`
}

func buildTriggerRequestJSON (trigData trigReqStruct, layerGroupID string) string {

	var trigObj TrigStruct
	trigObj.Name = trigData.Name
	trigObj.Enabled = true
	trigObj.Condition.EventTypeIDs = []string{trigData.EventTypeID}

	queryFilters := []QueryClause{}
	if trigData.SensorName != "" {
		sensorMatch := map[string]string{"SensorName":trigData.SensorName}
		queryFilters = append(queryFilters, QueryClause{ sensorMatch, nil })
	}
	if trigData.CloudCover != "" {
		cClause := CompClause{trigData.CloudCover, nil, ""}
		cloudRange := map[string]CompClause{"cloudCover":cClause}
		queryFilters = append(queryFilters, QueryClause{nil, cloudRange})
	}
	if trigData.MaxX != "" {
		cClause := CompClause{trigData.MaxX, nil, ""}
		XRange := map[string]CompClause{"MinX":cClause}
		queryFilters = append(queryFilters, QueryClause{nil, XRange})
	}
	if trigData.MinX != "" {
		cClause := CompClause{nil, trigData.MinX, ""}
		XRange := map[string]CompClause{"MaxX":cClause}
		queryFilters = append(queryFilters, QueryClause{nil, XRange})
	}
	if trigData.MaxY != "" {
		cClause := CompClause{trigData.MaxY, nil, ""}
		YRange := map[string]CompClause{"MinY":cClause}
		queryFilters = append(queryFilters, QueryClause{nil, YRange})
	}
	if trigData.MinY != "" {
		cClause := CompClause{nil, trigData.MinY, ""}
		YRange := map[string]CompClause{"MaxY":cClause}
		queryFilters = append(queryFilters, QueryClause{nil, YRange})
	}

	if trigData.MaxRes != "" || trigData.MinRes != "" {
		resClause := CompClause{nil, nil, ""}
		if trigData.MaxRes != "" {
			resClause.LTE = trigData.MaxRes
		}
		if trigData.MinRes != "" {
			resClause.GTE = trigData.MinRes
		}
		resFilter := map[string]CompClause{"resolution":resClause}
		queryFilters = append(queryFilters, QueryClause{nil, resFilter})
	}

	if trigData.MaxDate != "" || trigData.MinDate != "" {
		dateClause := CompClause{nil, nil, "yyyy-MM-dd'T'HH:mm:ssZZ"}
		if trigData.MaxDate != "" {
			dateClause.LTE = trigData.MaxDate
		}
		if trigData.MinDate != "" {
			dateClause.GTE = trigData.MinDate
		}
		dateFilter := map[string]CompClause{"acquiredDate":dateClause}
		queryFilters = append(queryFilters, QueryClause{nil, dateFilter})
	}

	trigObj.Condition.Query.Query.Bool.Filter = queryFilters


	trigObj.Job.JobType.Type = "execute-service"

	bfInpObj := &trigData.BFinpObj
	bfInpObj.LGroupID = layerGroupID
	bfInpObj.MetaURL = "$link"
	b, _ := json.Marshal(bfInpObj)

	jobInpObj := JobTypeInterface{ string(b), "text", "application/json" }
	jobOutpObj := JobTypeInterface{"", "text", "application/json"}
	jobIntMap := map[string]JobTypeInterface{"body":jobInpObj}
	trigObj.Job.JobType.Data = JobData{trigData.ServiceID, jobIntMap, []JobTypeInterface{jobOutpObj}}


	b2, _ := json.Marshal(trigObj)
	return string(b2)
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
