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
	//"errors"
	//"fmt"
	//"math"
	"net/http"
	//"os"
	//"strconv"
	//"strings"
	"testing"

	"github.com/venicegeo/pzsvc-lib"
)

// NewProductLine ....
func TestNewProductLine(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	testBodyStr := `{"name":what?}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	NewProductLine(w, &r)
	if *outInt < 300 && *outInt >= 200 {
		t.Error(`TestExecute: passed on what should have been a json failure.`)
	}
	*outStr = ""
	*outInt = 200

	testBodyStr = `{"bfInputJson": {"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.io/execute","pzAuthToken":"aaa","pzAddr":"https://pz-gateway.io","dbAuthToken":"aaaa","tideURL":"https://tideprediction.io/","bands":["coastal","swir1"],"metaDataURL":""},"cloudCover":10,"minDate":"2016-08-29","minx":0,"miny":0,"maxx":30,"maxy":30,"eventTypeId":"bbbbb","serviceId":"ccccc","name":"bf-handle test trigger"}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	cliOuts := []string{}

	pzsvc.SetMockClient(cliOuts, 200)

	NewProductLine(w, &r)
	if *outInt >= 300 || *outInt < 200 {
		t.Error(`TestNewProductLine: failed on what should have been a good run.  Error: ` + *outStr)
	}

}

// GetProductLines responds to a properly formed network request
// by sending out a list of triggers in JSON format.
func TestGetProductLines(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	testBodyStr := `{"name":what?}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	GetProductLines(w, &r)
	if *outInt < 300 && *outInt >= 200 {
		t.Error(`TestExecute: passed on what should have been a json failure.`)
	}

	testBodyStr = `{"eventTypeId":"bbbb","serviceId":"","createdBy":"","pzAuthToken":"49e2386b-b50b-491d-9949-37abcfc55264","pzAddr":"https://pz-gateway.geointservices.io"}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	cliOuts := []string{}

	pzsvc.SetMockClient(cliOuts, 200)
	GetProductLines(w, &r)
	if *outInt >= 300 || *outInt < 200 {
		t.Error(`TestGetProductLines: failed on what should have been a good run.  Error: ` + *outStr)
	}
}

func TestExtractTrigReqStruct(t *testing.T) {

	var triggerHolder pzsvc.Trigger

	json.Unmarshal([]byte(`{ "triggerId": "373e7f24-9bf2-4879-8dda-7d232f33fb7a", "name": "CI Testing Trigger", "eventTypeId": "34fbdba3-f638-43c6-8ff8-189b118165a1", "condition": { "query": { "bool": { "must": [ { "match": { "data~1476820914~dataType": "raster" } } ] } } }, "job": { "createdBy": "citester", "jobType": { "data": { "dataInputs": { "test": { "content": {}, "mimeType": "application/json", "type": "body" } }, "dataOutput": [ { "content": "filler text", "mimeType": "application/json", "type": "text" } ], "serviceId": "9998465f-644e-4fe1-bc78-49c68ec22173" }, "type": "execute-service" } }, "percolationId": "373e7f24-9bf2-4879-8dda-7d232f33fb7a", "createdBy": "citester", "createdOn": "2016-10-18T20:01:57.535270204Z", "enabled": true }`), triggerHolder)

	_, _ = extractTrigReqStruct(triggerHolder)

	json.Unmarshal([]byte(`{ "triggerId": "642594ee-7c8a-4061-84bf-b2d1e12e6b9b", "name": "CI Testing Trigger", "eventTypeId": "321733dd-a7a5-4699-847e-3bce0bf78ec0", "condition": { "query": { "bool": { "must": [ { "match": { "data~1477426129~dataType": "raster" } } ] } } }, "job": { "createdBy": "citester", "jobType": { "data": { "dataInputs": { "test": { "content": "{ \"log\": \"Received event with type $dataType\" }", "mimeType": "application/json", "type": "body" } }, "dataOutput": [ { "content": "filler text", "mimeType": "application/json", "type": "text" } ], "serviceId": "ad0fc512-bd0b-4bb2-80f3-d29ea7b948a4" }, "type": "execute-service" } }, "percolationId": "642594ee-7c8a-4061-84bf-b2d1e12e6b9b", "createdBy": "citester", "createdOn": "2016-10-25T20:08:53.950302328Z", "enabled": true }`), triggerHolder)
	_, _ = extractTrigReqStruct(triggerHolder)
}
func TestToString(t *testing.T) {
	var floatHolder float64
	var intHolder int
	var stringHolder string

	floatHolder = 1.234
	intHolder = 1
	stringHolder = "1"
	toString(floatHolder)
	toString(intHolder)
	toString(stringHolder)
}

func TestToFloat(t *testing.T) {
	var floatHolder float64
	var intHolder int
	var stringHolder string

	floatHolder = 1.234
	intHolder = 1
	stringHolder = "1.234"
	toFloat(floatHolder)
	toFloat(intHolder)
	toFloat(stringHolder)
}
