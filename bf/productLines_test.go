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
	//"encoding/json"
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

	testBodyStr = `{"eventTypeId":"bbbb","serviceId":"","createdBy":"","pzAuthToken":"aaa","pzAddr":"https://pz-gateway.io"}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	cliOuts := []string{}

	pzsvc.SetMockClient(cliOuts, 200)
	GetProductLines(w, &r)
	if *outInt >= 300 || *outInt < 200 {
		t.Error(`TestGetProductLines: failed on what should have been a good run.  Error: ` + *outStr)
	}
}

/*
Rough plan:
- put together a full product line creation call from postman data.
- send it through TestNewProductLine
- build a full get product lines call from postman data.
- send it through GetProductLines
be careful about the intermediary calls.  Some could get... messy.


*/
