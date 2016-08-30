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
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/venicegeo/pzsvc-lib"
)

type trigUIStruct struct {
	BFinpObj     gsInpStruct `json:"bfInputJSON,omitempty"`
	MaxX         float64     `json:"maxX"`
	MinX         float64     `json:"minX"`
	MaxY         float64     `json:"maxY"`
	MinY         float64     `json:"minY"`
	CloudCover   string      `json:"cloudCover"`
	MaxRes       string      `json:"maxRes,omitempty"`
	MinRes       string      `json:"minRes,omitempty"`
	MaxDate      string      `json:"maxDate"`
	MinDate      string      `json:"minDate"`
	SensorName   string      `json:"sensorName,omitempty"`
	SpatFilter   string      `json:"spatialFilterId"`
	EventTypeIDs []string    `json:"eventTypeId,omitempty"`
	ServiceID    string      `json:"serviceId,omitempty"`
	TriggerID    string      `json:"Id,omitempty"`
	CreatedBy    string      `json:"createdBy,omitempty"`
	Name         string      `json:"name,omitempty"`
}

func buildTriggerRequestJSON(trigData trigUIStruct, layerGID string) string {

	var trigObj pzsvc.Trigger
	trigObj.Name = trigData.Name
	trigObj.Enabled = true
	trigObj.Condition.EventTypeIDs = append(trigData.EventTypeIDs)

	queryFilters := []pzsvc.QueryClause{}
	if trigData.SensorName != "" {
		sensorMatch := map[string]string{"data.sensorName": trigData.SensorName}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: sensorMatch, Range: nil})
	}
	{
		cClause := pzsvc.CompClause{LTE: trigData.CloudCover, GTE: nil, Format: ""}
		cloudRange := map[string]pzsvc.CompClause{"data.cloudCover": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: cloudRange})
	}
	{
		cClause := pzsvc.CompClause{LTE: trigData.MaxX, GTE: nil, Format: ""}
		XRange := map[string]pzsvc.CompClause{"data.minX": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: XRange})
	}
	{
		cClause := pzsvc.CompClause{LTE: nil, GTE: trigData.MinX, Format: ""}
		XRange := map[string]pzsvc.CompClause{"data.maxX": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: XRange})
	}
	{
		cClause := pzsvc.CompClause{LTE: trigData.MaxY, GTE: nil, Format: ""}
		YRange := map[string]pzsvc.CompClause{"data.minY": cClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: YRange})
	}
	{
		cClause := pzsvc.CompClause{LTE: nil, GTE: trigData.MinY, Format: ""}
		YRange := map[string]pzsvc.CompClause{"data.maxY": cClause}
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
		resFilter := map[string]pzsvc.CompClause{"data.resolution": resClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: resFilter})
	}

	{
		dateClause := pzsvc.CompClause{LTE: nil, GTE: trigData.MinDate, Format: "yyyy-MM-dd'T'HH:mm:ssZZ"}
		if trigData.MaxDate != "" {
			dateClause.LTE = trigData.MaxDate
		}
		dateFilter := map[string]pzsvc.CompClause{"data.acquiredDate": dateClause}
		queryFilters = append(queryFilters, pzsvc.QueryClause{Match: nil, Range: dateFilter})
	}

	trigObj.Condition.Query.Query.Bool.Filter = queryFilters

	trigObj.Job.JobType.Type = "execute-service"

	bfInpObj := &trigData.BFinpObj
	bfInpObj.LGroupID = layerGID
	bfInpObj.MetaURL = "$link"
	b, _ := json.Marshal(bfInpObj)

	jobInpObj := pzsvc.DataType{Content: string(b), Type: "body", MimeType: "application/json"}
	jobOutpObj := pzsvc.DataType{Content: "", Type: "text", MimeType: "application/json"}
	jobIntMap := map[string]pzsvc.DataType{"body": jobInpObj}
	trigObj.Job.JobType.Data = pzsvc.JobData{ServiceID: trigData.ServiceID, DataInputs: jobIntMap, DataOutput: []pzsvc.DataType{jobOutpObj}}

	b2, _ := json.Marshal(trigObj)
	return string(b2)
}

// NewProductLine ....
func NewProductLine(w http.ResponseWriter, r *http.Request) {

	type outpType struct {
		TriggerID    string `json:"triggerId"`
		LayerGroupID string `json:"layerGroupId"`
	}

	type newTrigData struct {
		ID string `json:"triggerId"`
	}
	type newTrigOut struct {
		StatusCode int         `json:"statusCode"`
		Data       newTrigData `json:"data"`
	}

	inpObj := trigUIStruct{MinX: math.NaN(), MinY: math.NaN(), MaxX: math.NaN(), MaxY: math.NaN()}
	outpObj := outpType{}
	idObj := newTrigOut{}

	_, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}
	if math.IsNaN(inpObj.MinX + inpObj.MinY + inpObj.MaxX + inpObj.MaxY) {
		handleOut(w, "Error: Must specify full bounding box - minX, minY, maxX, and maxY.", nil, http.StatusBadRequest)
		return
	}
	if inpObj.MinDate == "" {
		handleOut(w, "Error: Must specify minDate.", nil, http.StatusBadRequest)
		return
	}
	if inpObj.CloudCover == "" {
		handleOut(w, "Error: Must specify cloudCover.", nil, http.StatusBadRequest)
		return
	}

	bfInpObj := &inpObj.BFinpObj

	if bfInpObj.PzAuth == "" {
		bfInpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if bfInpObj.DbAuth == "" {
		bfInpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	layerGID, err := pzsvc.AddGeoServerLayerGroup(bfInpObj.PzAddr, bfInpObj.PzAuth)
	if err != nil {
		handleOut(w, "Error: pzsvc.AddGeoServerLayerGroup: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}

	outJSON := buildTriggerRequestJSON(inpObj, layerGID)
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

	outpObj.LayerGroupID = layerGID

	b3, _ := json.Marshal(outpObj)
	fmt.Println(string(b3))

	handleOut(w, "", outpObj, http.StatusOK)
	fmt.Println("NewProductLine finished")
}

func extractTrigReqStruct(trigInp pzsvc.Trigger) (*trigUIStruct, error) {
	var trigOutp trigUIStruct

	trigOutp.Name = trigInp.Name
	trigOutp.TriggerID = trigInp.TriggerID
	trigOutp.EventTypeIDs = append(trigInp.Condition.EventTypeIDs)
	trigOutp.ServiceID = trigInp.Job.JobType.Data.ServiceID
	trigOutp.CreatedBy = trigInp.CreatedBy
	trigOutp.MinX = math.NaN()
	trigOutp.MaxX = math.NaN()
	trigOutp.MinY = math.NaN()
	trigOutp.MaxY = math.NaN()

	var bfInpObj gsInpStruct
	content := trigInp.Job.JobType.Data.DataInputs["body"].Content
	err := json.Unmarshal([]byte(content), &bfInpObj)
	if err != nil {
		return nil, errors.New(err.Error() + `  Initial input:` + content)
	}
	bfInpObj.MetaURL = ""
	trigOutp.BFinpObj = bfInpObj

	queryList := trigInp.Condition.Query.Query.Bool.Filter
	var query pzsvc.QueryClause
	for _, query = range queryList {
		var mKey, mVal, rKey string
		var rVal pzsvc.CompClause
		for mKey, mVal = range query.Match {
			switch mKey {
			case "data.sensorName":
				trigOutp.SensorName = mVal
			default:
			}
		}
		for rKey, rVal = range query.Range {
			switch rKey {
			case "data.cloudCover":
				trigOutp.CloudCover = toString(rVal.LTE)
			case "data.minX":
				trigOutp.MaxX, err = toFloat(rVal.LTE)
				if err != nil {
					return nil, errors.New(`extractTrigReqStruct: bad minX` + err.Error())
				}
			case "data.minY":
				trigOutp.MaxY, err = toFloat(rVal.LTE)
				if err != nil {
					return nil, errors.New(`extractTrigReqStruct: bad minY` + err.Error())
				}
			case "data.maxX":
				trigOutp.MinX, err = toFloat(rVal.GTE)
				if err != nil {
					return nil, errors.New(`extractTrigReqStruct: bad maxX` + err.Error())
				}
			case "data.maxY":
				trigOutp.MinY, err = toFloat(rVal.GTE)
				if err != nil {
					return nil, errors.New(`extractTrigReqStruct: bad maxY` + err.Error())
				}
			case "data.resolution":
				trigOutp.MaxRes = toString(rVal.LTE)
				trigOutp.MinRes = toString(rVal.GTE)
			case "data.acquiredDate":
				trigOutp.MaxDate = rVal.LTE.(string)
				trigOutp.MinDate = rVal.GTE.(string)
			default:
			}
		}
	}
	return &trigOutp, nil
}

func toString(input interface{}) string {
	switch inp := input.(type) {
	case int:
		return strconv.Itoa(inp)
	case float64:
		return strconv.FormatFloat(inp, 'E', -1, 64)
	case string:
		return inp
	default:
		return ""
	}
}

func toFloat(input interface{}) (float64, error) {
	switch inp := input.(type) {
	case int:
		return float64(inp), nil
	case float64:
		return inp, nil
	case string:
		return strconv.ParseFloat(inp, 64)
	default:
		return 0, errors.New("toFloat: not a valid type")
	}
}

// GetProductLines responds to a properly formed network request
// by sending out a list of triggers in JSON format.
func GetProductLines(w http.ResponseWriter, r *http.Request) {

	var inpObj struct {
		EventTypeID string `json:"eventTypeId"`
		ServiceID   string `json:"serviceId"`
		CreatedBy   string `json:"createdBy"`
		PzAddr      string `json:"pzAddr"`
		PzAuth      string `json:"pzAuthToken"`
		Order       string `json:"order"`
		SortBy      string `json:"sortBy"`
	}

	var outpObj struct {
		TrigList []trigUIStruct `json:"productLines"`
	}
	outpObj.TrigList = make([]trigUIStruct, 0)

	_, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error(), outpObj, http.StatusBadRequest)
		return
	}

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	//getJSON := `{"perPage":1000,"order":"desc","sortBy":"createdOn"}`

	var inTrigList pzsvc.TriggerList

	b, err := pzsvc.RequestKnownJSON("GET", "", inpObj.PzAddr+`/trigger?perPage=1000&order=desc&sortBy=createdOn`, inpObj.PzAuth, &inTrigList)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error()+".  http Error: "+string(b), outpObj, http.StatusInternalServerError)
		return
	}

AddTriggerLoop:
	for _, trig := range inTrigList.Data {
		if inpObj.EventTypeID != "" {
			onList := false
			for _, trigEvTyp := range trig.Condition.EventTypeIDs {
				if trigEvTyp == inpObj.EventTypeID {
					onList = true
				}
			}
			if !onList {
				continue AddTriggerLoop
			}
		}
		if inpObj.ServiceID != "" {
			if trig.Job.JobType.Data.ServiceID != inpObj.ServiceID {
				continue AddTriggerLoop
			}
		}
		if inpObj.CreatedBy != "" {
			if trig.CreatedBy != inpObj.CreatedBy {
				continue AddTriggerLoop
			}
		}
		if val, ok := trig.Job.JobType.Data.DataInputs["body"]; ok {
			if val.Content == "" {
				continue AddTriggerLoop
			}
		} else {
			continue AddTriggerLoop
		}
		var newTrig *trigUIStruct
		newTrig, err = extractTrigReqStruct(trig)
		if err != nil {
			fmt.Println(err.Error())
			continue AddTriggerLoop
		}
		trigFltTest := newTrig.MinX + newTrig.MinY + newTrig.MaxX + newTrig.MaxY
		if newTrig.CloudCover == "" || newTrig.MinDate == "" || math.IsNaN(trigFltTest) {
			fmt.Println(errors.New("Trigger not containing required parameter"))
			continue AddTriggerLoop
		}
		outpObj.TrigList = append(outpObj.TrigList, *newTrig)
	}

	b, err = json.Marshal(outpObj)
	if err != nil {
		handleOut(w, "Marshalling error: "+err.Error()+".", outpObj, http.StatusInternalServerError)
		return
	}
	pzsvc.HTTPOut(w, string(b), http.StatusOK)
	return
}
