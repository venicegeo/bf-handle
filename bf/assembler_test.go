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
	"github.com/paulsmith/gogeos/geos"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-lib"
	"net/http"
	"testing"
)

func TestAssembleShorelines(t *testing.T) {
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser(`{"name":what?}`)
	Execute(w, &r)
	*outStr = ""
	*outInt = 200
	AssembleShorelines(w, &r)
	Execute(w, &r)
	ExecuteBatch(w, &r)
	testBodyStr := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"","pzAddr":"https://pz-gateway.stage.geointservices.io","footprintsDataID":"1234","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}`

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
	Execute(w, &r)
	var mockMeta struct{ Data pzsvc.DataDesc }
	mockDataType := pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta := pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	mockMetaByts, _ := json.Marshal(mockMeta)

	cliOuts := []string{
		`{"lat":9, "lon":12, "dtg":"blah", "results":{"minimumTide24Hours":1.0,"maximumTide24Hours":5.0,"currentTide":3.0}}`,
		`{"InFiles":{"http://landsat_B1.TIF": "coastal", "http://landsat_B4.TIF": "swir1"}, "OutFiles":{"shoreline.geojson":"55"}, "HTTPStatus":200}`,
		string(mockMetaByts),
		string(mockFeatByts)}

	pzsvc.SetMockClient(cliOuts, 200)

	Execute(w, &r)
	AssembleShorelines(w, &r)
	ExecuteBatch(w, &r)
}
func TestExecuteBatch(t *testing.T) {
	var err error
	var mockMetaByts []byte
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
	Execute(w, &r)
	ExecuteBatch(w, &r)
	AssembleShorelines(w, &r)
	testBodyStr := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"","pzAddr":"https://pz-gateway.stage.geointservices.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/"}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr)
	// create and populate mock client here.
	Execute(w, &r)
	ExecuteBatch(w, &r)
	AssembleShorelines(w, &r)

	testBodyStr1 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"","baseline":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]},"footprintsDataID":"123","pzAddr":"https://pz-gateway.stage.geointservices.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr1)
	ExecuteBatch(w, &r)
	var mockMeta struct{ Data pzsvc.DataDesc }
	mockDataType := pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta := pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	if mockMetaByts, err = json.Marshal(mockMeta); err != nil {
		t.Error(err.Error())
	}
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
	AssembleShorelines(w, &r)

	testBodyStr2 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"123","baseline":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}],"footprintsDataID":"123","pzAddr":"https://pz-gateway.stage.geointservices.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}}`
	r.Body = pzsvc.GetMockReadCloser(testBodyStr2)
	ExecuteBatch(w, &r)
	mockDataType = pzsvc.DataType{Location: &pzsvc.FileLoc{FileSize: 500}}
	mockResMeta = pzsvc.ResMeta{Metadata: map[string]string{"prop1": "1", "prop2": "2"}}
	mockMeta.Data = pzsvc.DataDesc{DataID: "aaa", DataType: mockDataType, ResMeta: mockResMeta}
	if mockMetaByts, err = json.Marshal(mockMeta); err != nil {
		t.Error(err.Error())
	}
	cliOuts = []string{
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
	AssembleShorelines(w, &r)

	testBodyStr3 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"","baseline":{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.87088507656375,35.21515162500578]},"properties":{"name":"ABBOTTNEIGHBORHOODPARK","address":"1300SPRUCEST"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83775386582222,35.24980190252168]},"properties":{"name":"DOUBLEOAKSCENTER","address":"1326WOODWARDAV"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83827000459532,35.25674709224663]},"properties":{"name":"DOUBLEOAKSNEIGHBORHOODPARK","address":"2605DOUBLEOAKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83697759172735,35.25751734669229]},"properties":{"name":"DOUBLEOAKSPOOL","address":"1200NEWLANDRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.81647652154736,35.40148708491418]},"properties":{"name":"DAVIDB.WAYMERFLYINGREGIONALPARK","address":"15401HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Point","coordinates":[-80.83556459443902,35.39917224760999]},"properties":{"name":"DAVIDB.WAYMERCOMMUNITYPARK","address":"302HOLBROOKSRD"}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-80.72487831115721,35.26545403190955],[-80.72135925292969,35.26727607954368],[-80.71517944335938,35.26769654625573],[-80.7125186920166,35.27035945142482],[-80.70857048034668,35.268257165144064],[-80.70479393005371,35.268397319259996],[-80.70324897766113,35.26503355355979],[-80.71088790893555,35.2553619492954],[-80.71681022644043,35.2553619492954],[-80.7150936126709,35.26054831539319],[-80.71869850158691,35.26026797976481],[-80.72032928466797,35.26061839914875],[-80.72264671325684,35.26033806376283],[-80.72487831115721,35.26545403190955]]]},"properties":{"name":"PlazaRoadPark"}}]},"footprintsDataID":"123","pzAddr":"https://pz-gateway.stage.geointservices.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr3)
	Execute(w, &r)
	ExecuteBatch(w, &r)
	AssembleShorelines(w, &r)

	testBodyStr4 := `{"algoType":"pzsvc-ossim","svcURL":"https://pzsvc-ossim.stage.geointservices.io/execute","pzAuthToken":"123","Baseline":"","footprintsDataID":"","pzAddr":"https://pz-gateway.stage.geointservices.io","bandMergeType":"","bandMergeURL":"","tideURL":"https://bf-tideprediction.stage.geointservices.io/","dbAuthToken":"","bands":["coastal","swir1"],"metaDataJSON":{"type": "Feature","geometry": {"type": "Polygon","coordinates": [[35.0552646979563, -20.6249573123582],[36.814147099668, -20.9863928375569],[36.4165176126861, -22.719959598174],[34.6366754134012, -22.3522722379786],[35.0552646979563, -20.6249573123582]]},"properties": {"acquiredDate": "2016-06-18T07:36:07.536703+00:00","bands": {"blue": "http://landsat_B2.TIF","cirrus": "http://landsat_B9.TIF","coastal": "http://landsat_B1.TIF","green": "http://landsat_B3.TIF","nir": "http://landsat_B5.TIF","panchromatic": "http://landsat_B8.TIF","red": "http://landsat_B4.TIF","swir1": "http://landsat_B6.TIF","swir2": "http://landsat_B7.TIF","tirs1": "http://landsat_B10.TIF","tirs2": "http://landsat_B11.TIF"},"cloudCover": 8.6,"path": "http://landsat.com/index.html","resolution": 30,"sensorName": "Landsat8","thumb_large": "http://landsat_thumb_large.jpg","thumb_small": "http://landsat_thumb_small.jpg"},"id": "landsat:LC81660752016170LGN00","bbox": [34.6366754134012, -22.719959598174, 36.814147099668, -20.6249573123582]}}`

	r.Body = pzsvc.GetMockReadCloser(testBodyStr4)
	Execute(w, &r)
	ExecuteBatch(w, &r)
	AssembleShorelines(w, &r)
}

func TestForassembleShorelines(t *testing.T) {
	var asInpStrucHolder asInpStruct
	var geoCollectionHolder *geojson.FeatureCollection
	var mapHolder map[string]interface{}

	var bandsHolder []string

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{ "type": "FeatureCollection", "features": [ { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -47.63671875, -21.4121622297254 ], [ -47.63671875, 0.5273363048115169 ], [ -31.904296874999996, 0.5273363048115169 ], [ -31.904296874999996, -21.4121622297254 ], [ -47.63671875, -21.4121622297254 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -57.52441406249999, -11.092165893501988 ], [ -57.52441406249999, 8.53756535080403 ], [ -37.3095703125, 8.53756535080403 ], [ -37.3095703125, -11.092165893501988 ], [ -57.52441406249999, -11.092165893501988 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -70.048828125, 16.97274101999902 ], [ -70.4443359375, 8.363692651835823 ], [ -65.5224609375, 7.18810087117902 ], [ -60.6005859375, 9.88227549342994 ], [ -56.865234375, 16.214674588248542 ], [ -62.84179687499999, 20.3034175184893 ], [ -70.048828125, 16.97274101999902 ] ] ] } } ] }`))

	mapHolder = geoCollectionHolder.Map()
	bandsHolder = make([]string, 2)
	bandsHolder[0] = "costal"
	bandsHolder[1] = "swir1"
	asInpStrucHolder.AlgoType = "pzsvc-ossim"
	asInpStrucHolder.AlgoURL = "https://pzsvc-ossim.stage.geointservices.io/execute"
	asInpStrucHolder.Bands = bandsHolder
	asInpStrucHolder.Baseline = mapHolder
	asInpStrucHolder.Collections = geoCollectionHolder
	asInpStrucHolder.DbAuth = ""
	asInpStrucHolder.FootprintsDataID = "1234"
	asInpStrucHolder.ForceDetection = false
	asInpStrucHolder.JobName = "Test"
	asInpStrucHolder.LGroupID = "1234"
	asInpStrucHolder.PzAddr = "https://pz-gateway.stage.geointservices.io"
	asInpStrucHolder.PzAuth = ""
	asInpStrucHolder.SkipDetection = false
	asInpStrucHolder.TidesAddr = "https://bf-tideprediction.stage.geointservices.io"
	assembleShorelines(asInpStrucHolder)
	detectShorelines(asInpStrucHolder, geoCollectionHolder)

	asInpStrucHolder.SkipDetection = true
	assembleShorelines(asInpStrucHolder)
	detectShorelines(asInpStrucHolder, geoCollectionHolder)

}

func TestForfindBestMatchesAndgetBestScene(t *testing.T) {
	var line1, line2 *geos.Geometry
	var err error
	line1, err = geos.FromWKT("LINESTRING (0 0, 10 10, 20 20)")
	line2, err = geos.FromWKT("LINESTRING (5 0, 15 15, 17 17)")
	var geoCollectionHolder *geojson.FeatureCollection
	t.Log(err)
	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type": "FeatureCollection","features":[{"type": "Feature",   "properties": {},"geometry":{"type":"Polygon","coordinates":[[[-34.5,-7.0],[-35.5,-7.0],[-35.5,-6.0],[-34.5,-6.0],[-34.5,-7.0]]]}}]}`))
	findBestMatches(geoCollectionHolder, line1, line1)
	findBestMatches(geoCollectionHolder, line1, line2)
}

func TestForclipFootprintsAndupdateSceneTide(t *testing.T) {

	var geoCollectionHolder *geojson.FeatureCollection
	var geoFeatureArray []*geojson.Feature
	var line1, line2, poly1 *geos.Geometry
	var tide1 tideOut

	tide1.CurrTide = 3.13
	tide1.MaxTide = 3.45
	tide1.MinTide = 2.95
	line1, _ = geos.FromWKT("LINESTRING (-34.9 6.5, -36.9 6.5, -35.5 7)")
	line2, _ = geos.FromWKT("LINESTRING (5 0, 15 15, 17 17)")

	poly1, _ = geos.FromWKT("POLYGON((-44.505615234376 -2.8564453125005, -32.288818359376 -2.9443359375005, -31.937255859376 -18.9404296875, -51.185302734376 -17.7978515625, -51.273193359376 -17.7099609375, -44.505615234376 -2.8564453125005))")

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{ "type": "FeatureCollection", "features": [ { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -47.63671875, -21.4121622297254 ], [ -47.63671875, 0.5273363048115169 ], [ -31.904296874999996, 0.5273363048115169 ], [ -31.904296874999996, -21.4121622297254 ], [ -47.63671875, -21.4121622297254 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -57.52441406249999, -11.092165893501988 ], [ -57.52441406249999, 8.53756535080403 ], [ -37.3095703125, 8.53756535080403 ], [ -37.3095703125, -11.092165893501988 ], [ -57.52441406249999, -11.092165893501988 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -70.048828125, 16.97274101999902 ], [ -70.4443359375, 8.363692651835823 ], [ -65.5224609375, 7.18810087117902 ], [ -60.6005859375, 9.88227549342994 ], [ -56.865234375, 16.214674588248542 ], [ -62.84179687499999, 20.3034175184893 ], [ -70.048828125, 16.97274101999902 ] ] ] } } ] }`))

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{ "type": "FeatureCollection", "features": [ {"type":"Feature","geometry":{"coordinates":[[-41.68380384,-3.86901559],[-41.68344951,-3.86733807],[-41.68361042,-3.86726774],[-41.68384764,-3.86719616],[-41.68413582,-3.86716065],[-41.68444963,-3.86719857],[-41.68476372,-3.86734723],[-41.68505276,-3.86764398],[-41.68529141,-3.86812615],[-41.68537007,-3.86836772],[-41.68542289,-3.8685737],[-41.68544916,-3.86875115],[-41.68544817,-3.86890711],[-41.6854192,-3.86904862],[-41.68536156,-3.86918273],[-41.68527452,-3.86931649],[-41.68515738,-3.86945693],[-41.68495458,-3.86964114],[-41.68475013,-3.86975328],[-41.68454967,-3.8697952],[-41.68435881,-3.86976873],[-41.68418317,-3.86967571],[-41.68402839,-3.86951795],[-41.68390007,-3.8692973],[-41.68380384,-3.86901559]],"type":"LineString"},"properties":{"24hrMaxTide":"4.272558868170382","24hrMinTide":"2.4257490639311676","algoCmd":"ossim-cli shoreline --image img1.TIF,img2.TIF --projection geo-scaled --prop 24hrMinTide:2.4257490639311676 --prop resolution:30 --prop classification:Unclassified --prop dataUsage:Not_to_be_used_for_navigational_or_targeting_purposes. --prop sensorName:Landsat8 --prop 24hrMaxTide:4.272558868170382 --prop currentTide:3.4136017245233523 --prop sourceID:landsat:LC82190622016285LGN00 --prop dateTimeCollect:2016-10-11T12:59:05.157475+00:00 shoreline.geojson","algoName":"BF_Algo_NDWI","algoProcTime":"20161031.133058.4026","algoVersion":"0.0","classification":"Unclassified","currentTide":"3.4136017245233523","dataUsage":"Not_to_be_used_for_navigational_or_targeting_purposes.","dateTimeCollect":"2016-10-11T12:59:05.157475+00:00","resolution":"30","sensorName":"Landsat8","sourceID":"landsat:LC82190622016285LGN00"}} ] }`))

	geoFeatureArray = geoCollectionHolder.Features
	_ = clipFootprints(geoFeatureArray, line1)
	_ = clipFootprints(geoFeatureArray, line2)
	_ = clipFootprints(geoFeatureArray, poly1)

	for _, feature := range geoFeatureArray {
		updateSceneTide(feature, tide1)
	}

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type": "FeatureCollection","features":[{"type":"Feature","geometry":{"coordinates":[[-41.68380384,-3.86901559],[-41.68344951,-3.86733807],[-41.68361042,-3.86726774],[-41.68384764,-3.86719616],[-41.68413582,-3.86716065],[-41.68444963,-3.86719857],[-41.68476372,-3.86734723],[-41.68505276,-3.86764398],[-41.68529141,-3.86812615],[-41.68537007,-3.86836772],[-41.68542289,-3.8685737],[-41.68544916,-3.86875115],[-41.68544817,-3.86890711],[-41.6854192,-3.86904862],[-41.68536156,-3.86918273],[-41.68527452,-3.86931649],[-41.68515738,-3.86945693],[-41.68495458,-3.86964114],[-41.68475013,-3.86975328],[-41.68454967,-3.8697952],[-41.68435881,-3.86976873],[-41.68418317,-3.86967571],[-41.68402839,-3.86951795],[-41.68390007,-3.8692973],[-41.68380384,-3.86901559]],"type":"LineString"},"properties":{"24hrMaxTide":"4.272558868170382","24hrMinTide":"2.4257490639311676","algoCmd":"ossim-cli shoreline --image img1.TIF,img2.TIF --projection geo-scaled --prop 24hrMinTide:2.4257490639311676 --prop resolution:30 --prop classification:Unclassified --prop dataUsage:Not_to_be_used_for_navigational_or_targeting_purposes. --prop sensorName:Landsat8 --prop 24hrMaxTide:4.272558868170382 --prop currentTide:3.4136017245233523 --prop sourceID:landsat:LC82190622016285LGN00 --prop dateTimeCollect:2016-10-11T12:59:05.157475+00:00 shoreline.geojson","algoName":"BF_Algo_NDWI","algoProcTime":"20161031.133058.4026","algoVersion":"0.0","classification":"Unclassified","currentTide":"3.4136017245233523","dataUsage":"Not_to_be_used_for_navigational_or_targeting_purposes.","dateTimeCollect":"2016-10-11T12:59:05.157475+00:00","resolution":"30","sensorName":"Landsat8","sourceID":"landsat:LC82190622016285LGN00"}}]}`))

	//for _, feature := range geoFeatureArray {
	//updateSceneTide(feature, tide1)
	//}

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type": "FeatureCollection","features":[{"type": "Feature",   "properties": {},"geometry":{"type":"Polygon","coordinates":[[[-34.5,-7.0],[-35.5,-7.0],[-35.5,-6.0],[-34.5,-6.0],[-34.5,-7.0]]]}}]}`))

	for _, feature := range geoFeatureArray {
		updateSceneTide(feature, tide1)
	}

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type": "FeatureCollection","features":[{"type":"Feature","geometry":{"coordinates":[[-41.68380384,-3.86901559],[-41.68344951,-3.86733807],[-41.68361042,-3.86726774],[-41.68384764,-3.86719616],[-41.68413582,-3.86716065],[-41.68444963,-3.86719857],[-41.68476372,-3.86734723],[-41.68505276,-3.86764398],[-41.68529141,-3.86812615],[-41.68537007,-3.86836772],[-41.68542289,-3.8685737],[-41.68544916,-3.86875115],[-41.68544817,-3.86890711],[-41.6854192,-3.86904862],[-41.68536156,-3.86918273],[-41.68527452,-3.86931649],[-41.68515738,-3.86945693],[-41.68495458,-3.86964114],[-41.68475013,-3.86975328],[-41.68454967,-3.8697952],[-41.68435881,-3.86976873],[-41.68418317,-3.86967571],[-41.68402839,-3.86951795],[-41.68390007,-3.8692973],[-41.68380384,-3.86901559]],"type":"LineString"},"properties":{"24hrMaxTide":"4.272558868170382","24hrMinTide":"2.4257490639311676","algoCmd":"ossim-cli shoreline --image img1.TIF,img2.TIF --projection geo-scaled --prop 24hrMinTide:2.4257490639311676 --prop resolution:30 --prop classification:Unclassified --prop dataUsage:Not_to_be_used_for_navigational_or_targeting_purposes. --prop sensorName:Landsat8 --prop 24hrMaxTide:4.272558868170382 --prop currentTide:3.4136017245233523 --prop sourceID:landsat:LC82190622016285LGN00 --prop dateTimeCollect:2016-10-11T12:59:05.157475+00:00 shoreline.geojson","algoName":"BF_Algo_NDWI","algoProcTime":"20161031.133058.4026","algoVersion":"0.0","classification":"Unclassified","currentTide":"3.4136017245233523","dataUsage":"Not_to_be_used_for_navigational_or_targeting_purposes.","dateTimeCollect":"2016-10-11T12:59:05.157475+00:00","resolution":"30","sensorName":"Landsat8","sourceID":"landsat:LC82190622016285LGN00"}}]}`))

	geoFeatureArray = geoCollectionHolder.Features
	_ = clipFootprints(geoFeatureArray, line1)
	_ = clipFootprints(geoFeatureArray, line2)
	_ = clipFootprints(geoFeatureArray, poly1)

	for _, feature := range geoFeatureArray {
		updateSceneTide(feature, tide1)
	}

	//for _, feature := range geoFeatureArray {
	//	updateSceneTide(feature, tide1)
	//}

	for _, feature := range geoFeatureArray {
		updateSceneTide(feature, tide1)
	}

}
func TestForselfClipAndtoTidesIn(t *testing.T) {

	var geoCollectionHolder *geojson.FeatureCollection
	var geoFeatureArray []*geojson.Feature

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{ "type": "FeatureCollection", "features": [ { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -47.63671875, -21.4121622297254 ], [ -47.63671875, 0.5273363048115169 ], [ -31.904296874999996, 0.5273363048115169 ], [ -31.904296874999996, -21.4121622297254 ], [ -47.63671875, -21.4121622297254 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -57.52441406249999, -11.092165893501988 ], [ -57.52441406249999, 8.53756535080403 ], [ -37.3095703125, 8.53756535080403 ], [ -37.3095703125, -11.092165893501988 ], [ -57.52441406249999, -11.092165893501988 ] ] ] } }, { "type": "Feature", "properties": {}, "geometry": { "type": "Polygon", "coordinates": [ [ [ -70.048828125, 16.97274101999902 ], [ -70.4443359375, 8.363692651835823 ], [ -65.5224609375, 7.18810087117902 ], [ -60.6005859375, 9.88227549342994 ], [ -56.865234375, 16.214674588248542 ], [ -62.84179687499999, 20.3034175184893 ], [ -70.048828125, 16.97274101999902 ] ] ] } } ] }`))
	geoFeatureArray = geoCollectionHolder.Features
	_ = selfClip(geoFeatureArray)
	toTidesIn(geoFeatureArray)

	geoCollectionHolder, _ = geojson.FeatureCollectionFromBytes([]byte(`{"type": "FeatureCollection","features":[{"type": "Feature",   "properties": {},"geometry":{"type":"Polygon","coordinates":[[[-34.5,-7.0],[-35.5,-7.0],[-35.5,-6.0],[-34.5,-6.0],[-34.5,-7.0]]]}}]}`))
	geoFeatureArray = geoCollectionHolder.Features
	_ = selfClip(geoFeatureArray)
	toTidesIn(geoFeatureArray)

}
