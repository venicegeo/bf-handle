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
	"log"
	"math"
	"time"

	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

type tideIn struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	Dtg string  `json:"dtg"`
}

type tidesIn struct {
	Locations []tideIn                    `json:"locations"`
	Map       map[string]*geojson.Feature `json:"-"`
}

type tideOut struct {
	MinTide  float64 `json:"minimumTide24Hours"`
	MaxTide  float64 `json:"maximumTide24Hours"`
	CurrTide float64 `json:"currentTide"`
}

type tideWrapper struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Dtg     string  `json:"dtg"`
	Results tideOut `json:"results"`
}
type tidesOut struct {
	Locations []tideWrapper `json:"locations"`
}

func findTide(bbox geojson.BoundingBox, timeStr string) *tideIn {
	var (
		center  *geojson.Point
		dtgTime time.Time
		err     error
	)
	if center = bbox.Centroid(); center == nil {
		return nil
	}
	if dtgTime, err = time.Parse("2006-01-02T15:04:05.000000-07:00", timeStr); err != nil {
		return nil
	}
	return &tideIn{Lat: center.Coordinates[1], Lon: center.Coordinates[0], Dtg: dtgTime.Format("2006-01-02-15-04")}
}

func toTideIn(feature *geojson.Feature) *tideIn {
	return findTide(feature.Bbox, feature.PropertyString("acquiredDate"))
}

func toTidesIn(features []*geojson.Feature) *tidesIn {
	var (
		result     tidesIn
		currTideIn *tideIn
	)
	result.Map = make(map[string]*geojson.Feature)
	for _, feature := range features {
		if feature.PropertyFloat("CurrentTide") != math.NaN() {
			if currTideIn = toTideIn(feature); currTideIn == nil {
				log.Print(pzsvc.TraceStr(`Could not get tide information from feature ` + feature.IDStr() + ` because required elements did not exist.`))
				continue
			}
			result.Locations = append(result.Locations, *currTideIn)
			result.Map[currTideIn.Dtg] = feature
		}
	}
	switch len(result.Locations) {
	case 0:
		return nil
	default:
		return &result
	}
}

func getTide(inpObj tideIn, tideAddr string) (*tideOut, error) {
	var outpObj tideOut
	byts, err := json.Marshal(inpObj)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	_, err = pzsvc.RequestKnownJSON("POST", string(byts), tideAddr, "", &outpObj)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	return &outpObj, nil
}

func getTides(inpObj tidesIn, tideAddr string) (*tidesOut, error) {
	var (
		outpObj tidesOut
	)
	bytes, err := json.Marshal(inpObj)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	if _, err = pzsvc.RequestKnownJSON("POST", string(bytes), tideAddr, "", &outpObj); err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	return &outpObj, nil
}

func updateSceneTide(scene *geojson.Feature, inpObj tideOut) {
	properties := make(map[string]interface{})
	properties["CurrTide"] = inpObj.CurrTide
	properties["24hrMinTide"] = inpObj.MinTide
	properties["24hrMaxTide"] = inpObj.MaxTide

	if err := catalog.SaveFeatureProperties(scene.IDStr(), properties); err != nil {
		log.Print(pzsvc.TraceStr("Failed to update feature " + scene.IDStr() + " with tide information: " + err.Error()))
	}
}
