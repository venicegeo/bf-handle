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
	//	"errors"
	//  "fmt"
	"net/http"
	//	"os"
	//	"strconv"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-lib"
	"testing"
)

func TestAssembleShorelines(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser(`{"name":what?}`)
	Execute(w, &r)
	if *outInt < 300 && *outInt >= 200 {
		t.Error(`TestExecute: passed on what should have been a json failure.`)
	}
	*outStr = ""
	*outInt = 200
	AssembleShorelines(w, &r)
	testBodyStr := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim/execute","pzAuthToken":"","pzAddr":"https://pz-gateway.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	// create and populate mock client here.

	mockProps := map[string]interface{}{"acquiredDate": "today", "sensorName": "2", "resolution": 3, "classification": "UNCLASSIFIED"}
	mockFeat := geojson.Feature{ID: "aaaa", Properties: mockProps}
	mockFeats := []*geojson.Feature{&mockFeat}
	mockFeatColl := geojson.FeatureCollection{Features: mockFeats}
	mockFeatByts, err := json.Marshal(mockFeatColl)
	if err != nil {
		t.Error(`TestExecute: failed to marshal dummy data.  What's wrong with you?`)
	}
	AssembleShorelines(w, &r)
	var mockMeta struct{ Data pzsvc.DataDesc }
	mockDataType := pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta := pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	mockMetaByts, _ := json.Marshal(mockMeta)

	cliOuts := []string{
		`{"lat":9, "lon":12, "dtg":"blah", "results":{"minimumTide24Hours":1.0,"maximumTide24Hours":5.0,"currentTide":3.0}}`,
		`{"InFiles":{"http://landsat_B1.TIF": "coastal", "http://landsat_B4.TIF": "swir1"}, "OutFiles":{"shoreline.geojson":"55"}, "HTTPStatus":200}`,
		string(mockMetaByts),
		string(mockFeatByts),
		`{"data":{"jobId":"aaaa"}}`,
		`{"data":{"status":"Success","result":{"dataId":"aaaa"}}}`,
		`{"data":{"jobId":"aaaa"}}`,
		`{"data":{"status":"Success","result":{"deployment":{"deploymentId":"aaaa","dataId":"aaa"}}}}`}

	pzsvc.SetMockClient(cliOuts, 200)

	Execute(w, &r)
	AssembleShorelines(w, &r)
}
func TestExecuteBatch(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser(`{"name":what?}`)
	Execute(w, &r)
	if *outInt < 300 && *outInt >= 200 {
		t.Error(`TestExecute: passed on what should have been a json failure.`)
	}
	*outStr = ""
	*outInt = 200

	mockFeatByts := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]}`)
	ExecuteBatch(w, &r)
	testBodyStr := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim/execute","pzAuthToken":"","pzAddr":"https://pz-gateway.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	// create and populate mock client here.
	ExecuteBatch(w, &r)
	testBodyStr2 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim/execute","pzAuthToken":"","baseline":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]},"pzAddr":"https://pz-gateway.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","footprintsDataID":"12345","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr2)
	ExecuteBatch(w, &r)
	var mockMeta struct{ Data pzsvc.DataDesc }
	mockDataType := pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta := pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	mockMetaByts, _ := json.Marshal(mockMeta)
	cliOuts := []string{
		`{"lat":9, "lon":12, "dtg":"blah", "results":{"minimumTide24Hours":1.0,"maximumTide24Hours":5.0,"currentTide":3.0}}`,
		`{"InFiles":{"http://landsat_B1.TIF": "coastal", "http://landsat_B4.TIF": "swir1"}, "OutFiles":{"shoreline.geojson":"55"}, "HTTPStatus":200}`,
		string(mockMetaByts),
		string(mockFeatByts),
		`{"data":{"jobId":"aaaa"}}`,
		`{"data":{"status":"Success","result":{"dataId":"aaaa"}}}`,
		`{"data":{"jobId":"aaaa"}}`,
		`{"data":{"status":"Success","result":{"deployment":{"deploymentId":"aaaa","dataId":"aaa"}}}}`}
	pzsvc.SetMockClient(cliOuts, 200)

	Execute(w, &r)
	ExecuteBatch(w, &r)

	testBodyStr3 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim/execute","pzAuthToken":"","baseline":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]},"FootprintsDataID":"1234","pzAddr":"https://pz-gateway.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","footprintsDataID":"12345","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr3)
	ExecuteBatch(w, &r)
	mockDataType = pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta = pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	mockMetaByts, _ = json.Marshal(mockMeta)

	pzsvc.SetMockClient(cliOuts, 200)

	Execute(w, &r)
	ExecuteBatch(w, &r)

	ExecuteBatch(w, &r)

	mockDataType = pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta = pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	mockMetaByts, _ = json.Marshal(mockMeta)
	pzsvc.SetMockClient(cliOuts, 200)

	Execute(w, &r)
	ExecuteBatch(w, &r)
}

func TestDetectShorelines(t *testing.T) {
	var asInpStrucHolder asInpStruct
	var geoCollectionHolder *geojson.FeatureCollection

	var bandsHolder []string

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]}`))
	bandsHolder = make([]string, 2)
	bandsHolder[0] = "costal"
	bandsHolder[1] = "swir1"
	asInpStrucHolder.AlgoType = "pzsvc-ossim"
	asInpStrucHolder.AlgoURL = "https://pzsvc-ossim/execute"
	asInpStrucHolder.Bands = bandsHolder
	asInpStrucHolder.Baseline = nil
	asInpStrucHolder.Collections = geoCollectionHolder
	asInpStrucHolder.DbAuth = ""
	asInpStrucHolder.FootprintsDataID = "1234"
	asInpStrucHolder.ForceDetection = false
	asInpStrucHolder.JobName = "Test"
	asInpStrucHolder.LGroupID = "1234"
	asInpStrucHolder.PzAddr = "https://pz-gateway.io"
	asInpStrucHolder.PzAuth = ""
	asInpStrucHolder.SkipDetection = false
	asInpStrucHolder.TidesAddr = "https://bf-tideprediction.stage.geointservices.io"

	asInpStrucHolder.SkipDetection = false
	asInpStrucHolder.TidesAddr = ""

	detectShorelines(asInpStrucHolder, geoCollectionHolder)
}