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
	//	"errors"
	//  "fmt"
	"net/http"
	//	"os"
	//	"strconv"
	//"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-lib"
	"testing"
)

func TestResultsByScene(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser(`{"name":what?}`)
	Execute(w, &r)
	if *outInt < 300 && *outInt >= 200 {
		t.Error(`TestResultsByScene: passed on what should have been a json failure.`)
	}
	*outStr = ""
	*outInt = 200

	testBodyStr := `{"imageId":"landsat:LC81130812016183LGN00","pzAuthToken":"aaaa","pzAddr":"https://pz-gateway.io"}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)

	cliOuts := []string{}

	pzsvc.SetMockClient(cliOuts, 200)

	ResultsByScene(w, &r)
	if *outInt >= 300 || *outInt < 200 {
		t.Error(`TestResultsByScene: failed on what should have been a good run.  Error: ` + *outStr)
	}
}
