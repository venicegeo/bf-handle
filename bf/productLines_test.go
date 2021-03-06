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

	testBodyStr = `{"eventTypeId":"bbbb","serviceId":"123","createdBy":"TommyG","pzAuthToken":"49e2386b-b50b-491d-9949-37abcfc55264","pzAddr":"https://pz-gateway.geointservices.io"}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	cliOuts := []string{}

	pzsvc.SetMockClient(cliOuts, 200)
	GetProductLines(w, &r)
	if *outInt >= 300 || *outInt < 200 {
		t.Error(`TestGetProductLines: failed on what should have been a good run.  Error: ` + *outStr)
	}

	testBodyStr = `{"name":"bf-handle test trigger","enabled":true,"eventTypeId":"bbbbb","condition":{"query":{"bool":{"filter":[{"range":{"data.cloudCover":{"lte":10}}},{"range":{"data.minx":{"lte":30}}},{"range":{"data.maxx":{"gte":0}}},{"range":{"data.miny":{"lte":30}}},{"range":{"data.maxy":{"gte":0}}},{"range":{"data.acquiredDate":{"gte":"2016-08-29","format":"yyyy-MM-dd'T'HH:mm:ssZZ"}}}]}}},"job":{"jobType":{"type":"execute-service","data":{"serviceId":"ccccc","dataInputs":{"body":{"content":"{\"algoType\":\"pzsvc-ossim\",\"svcURL\":\"https://pzsvc-ossim.io/execute\",\"tideURL\":\"https://tideprediction.io/\",\"metaDataURL\":\"$link\",\"bands\":[\"coastal\",\"swir1\"],\"pzAuthToken\":\"aaa\",\"pzAddr\":\"https://pz-gateway.io\",\"dbAuthToken\":\"aaaa\",\"lGroupId\":\"\",\"jobName\":\"\"}","type":"body","mimeType":"application/json"}},"dataOutput":[{"type":"text","mimeType":"application/json"}]}}}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	cliOuts = []string{}

	pzsvc.SetMockClient(cliOuts, 200)
	GetProductLines(w, &r)

}

func TestExtractTrigReqStruct(t *testing.T) {

	var triggerHolder pzsvc.Trigger

	json.Unmarshal([]byte(`{ "triggerId": "ea9a6b00-0316-4b1f-9467-f1fa913aef2a", "name": "Beachfront Recurring Harvest", "eventTypeId": "f9315fe1-a591-4553-a5f5-dc99fc88b0ba", "condition": { "query": { "bool": { "filter": [{"range":{"data~data~cloudCover":{"lte":10}}},{"range":{"data~data~minx":{"lte":30}}},{"range":{"data~data~maxx":{"gte":0}}},{"range":{"data~data~miny":{"lte":30}}},{"range":{"data~data~maxy":{"gte":0}}},{"range":{"data~data~acquiredDate":{"gte":"2016-08-29","format":"yyyy-MM-dd'T'HH:mm:ssZZ"}}}]} } }, "job": { "createdBy": "yutzlejp", "jobType": { "data": { "dataInputs": { "body": { "content":"{\"algoType\":\"pzsvc-ossim\", \"svcURL\":\"https://pzsvc-ossim.int.geointservices.io/execute\", \"pzAuthToken\":\"==\", \"pzAddr\":\"https://pz-gateway.stage.geointservices.io\", \"bandMergeType\":\"pzsvc-ossim\", \"bandMergeURL\":\"https://pzsvc-ossim.int.geointservices.io/execute\", \"tideURL\":\"https://bf-tideprediction.stage.geointservices.io\", \"dbAuthToken\":\"ea28a0b4396b4c20b9d62760ce757261\", \"bands\":[\"coastal\",\"swir1\"], \"metaDataURL\":\"http://pzsvc-image-catalog.int.geointservices.io/image/landsat:LC80900892015290LGN00\" }", "mimeType": "application/json", "type": "body" } }, "dataOutput": [ { "content": null, "mimeType": "text/plain", "type": "text" } ], "serviceId": "344f59c7-ea74-4727-ae21-2b89eb9e17cc" }, "type": "execute-service" } }, "percolationId": "ea9a6b00-0316-4b1f-9467-f1fa913aef2a", "createdBy": "yutzlejp", "createdOn": "2016-10-18T21:13:52.491407527Z", "enabled": true}`), &triggerHolder)
	t.Log(triggerHolder.Name)
	t.Log(triggerHolder.Job.JobType.Data.DataInputs["body"].Content)

	_, _ = extractTrigReqStruct(triggerHolder)

	json.Unmarshal([]byte(`{ "triggerId": "373e7f24-9bf2-4879-8dda-7d232f33fb7a", "name": "CI Testing Trigger", "eventTypeId": "34fbdba3-f638-43c6-8ff8-189b118165a1", "condition": { "query": { "bool": { "must": [ { "match": { "data~1476820914~dataType": "raster" } } ] } } }, "job": { "createdBy": "citester", "jobType": { "data": { "dataInputs": { "test": { "content": {}, "mimeType": "application/json", "type": "body" } }, "dataOutput": [ { "content": "filler text", "mimeType": "application/json", "type": "text" } ], "serviceId": "9998465f-644e-4fe1-bc78-49c68ec22173" }, "type": "execute-service" } }, "percolationId": "373e7f24-9bf2-4879-8dda-7d232f33fb7a", "createdBy": "citester", "createdOn": "2016-10-18T20:01:57.535270204Z", "enabled": true }`), triggerHolder)

	_, _ = extractTrigReqStruct(triggerHolder)

	json.Unmarshal([]byte(`{ "triggerId": "642594ee-7c8a-4061-84bf-b2d1e12e6b9b", "name": "CI Testing Trigger", "eventTypeId": "321733dd-a7a5-4699-847e-3bce0bf78ec0", "condition": { "query": { "bool": { "must": [ { "match": { "data~1477426129~dataType": "raster" } } ] } } }, "job": { "createdBy": "citester", "jobType": { "data": { "dataInputs": { "test": { "content": "{ \"log\": \"Received event with type $dataType\" }", "mimeType": "application/json", "type": "body" } }, "dataOutput": [ { "content": "filler text", "mimeType": "application/json", "type": "text" } ], "serviceId": "ad0fc512-bd0b-4bb2-80f3-d29ea7b948a4" }, "type": "execute-service" } }, "percolationId": "642594ee-7c8a-4061-84bf-b2d1e12e6b9b", "createdBy": "citester", "createdOn": "2016-10-25T20:08:53.950302328Z", "enabled": true }`), triggerHolder)
	_, _ = extractTrigReqStruct(triggerHolder)

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
