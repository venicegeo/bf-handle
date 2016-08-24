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
	"log"
	"net/http"

	"github.com/paulsmith/gogeos/geos"
	"github.com/venicegeo/geojson-geos-go/geojsongeos"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-lib"
)

type asInpStruct struct {
	Collections []gsOutpStruct         `json:"collections"` // Slice of collection objects
	PzAuth      string                 `json:"pzAuthToken"` // Auth string for this Pz instance
	PzAddr      string                 `json:"pzAddr"`      // gateway URL for this Pz instance
	Baseline    map[string]interface{} `json:"baseline"`    // Baseline shoreline, as GeoJSON
	// DbAuth      string         `json:"dbAuthToken"`   // Auth string for the initial image database
	// LGroupID    string         `json:"lGroupId"`      // UUID string for the target geoserver layer group
	// JobName     string         `json:"resultName"`    // Arbitrary user-defined string to aid in later reference
}

// AssembleShorelines creates a single dataset from some input or something
func AssembleShorelines(w http.ResponseWriter, r *http.Request) {
	var (
		b             []byte
		err           error
		inpObj        asInpStruct
		outpObj       gsOutpStruct
		gjBaseline    interface{}
		gjResult      interface{}
		geosBaseline  *geos.Geometry
		assembledGeom *geos.Geometry
	)

	// clients to this function expect a JSON response
	// containing the error message
	handleError := func(errmsg string, status int) {
		outpObj.Error = errmsg
		if b, err = json.Marshal(outpObj); err != nil {
			b = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		http.Error(w, string(b), status)
	}

	if b, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		tracedError := pzsvc.TracedError("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(b))
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	gjBaseline = geojson.FromMap(inpObj.Baseline)

	if geosBaseline, err = geojsongeos.GeosFromGeoJSON(gjBaseline); err != nil {
		tracedError := pzsvc.TracedError("Could not convert GeoJSON object to GEOS geometry: " + err.Error())
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	if assembledGeom = assembleShorelines(inpObj, geosBaseline); assembledGeom == nil {
		w.Write([]byte("Found nothing. Sorry"))
	} else {
		if gjResult, err = geojsongeos.GeoJSONFromGeos(assembledGeom); err != nil {
			tracedError := pzsvc.TracedError("Could not convert output GEOS geometry to GeoJSON object: " + err.Error())
			handleError(tracedError.Error(), http.StatusInternalServerError)
			return
		}
		if b, err = geojson.Write(gjResult); err != nil {
			tracedError := pzsvc.TracedError("Failed to write output GeoJSON object: " + err.Error())
			handleError(tracedError.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(b)
	}
}

func assembleShorelines(inpObj asInpStruct, baseline *geos.Geometry) *geos.Geometry {
	var (
		gjIfc interface{}
		collGeom,
		collGeomPart,
		foundGeom,
		intersectGeom,
		clippedGeom *geos.Geometry
		b      []byte
		err    error
		fc     *geojson.FeatureCollection
		ok     bool
		empty  bool
		result *geos.Geometry
		foundGeoms,
		clippedGeoms []*geos.Geometry
		count int
	)

	log.Printf("baseline: %v", baseline.String())

	for _, collection := range inpObj.Collections {
		clippedGeoms = nil

		if collGeom, err = geojsongeos.GeosFromGeoJSON(geojson.FromMap(collection.Geometry.(map[string]interface{}))); err != nil {
			log.Printf("%T", collection.Geometry)
			log.Printf(pzsvc.TracedError("Could not convert GeoJSON object to GEOS geometry: " + err.Error()).Error())
			continue
		}

		// Because this can't be easy, the intersection function doesn't work well with multipolygons.
		// We have to split them apart and test them individually
		if count, err = collGeom.NGeometry(); err != nil {
			log.Printf(pzsvc.TracedError("Could not count the parts of the collected geometry: " + err.Error()).Error())
			log.Printf("collGeom: %v", collGeom.String())
			continue
		}

		for inx := 0; inx < count; inx++ {
			collGeomPart, _ = collGeom.Geometry(inx)

			if clippedGeom, err = baseline.Intersection(collGeomPart); err != nil {
				log.Printf(pzsvc.TracedError("Could not clip the baseline geometry: " + err.Error()).Error())
				log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			}

			if empty, err = clippedGeom.IsEmpty(); err != nil {
				log.Printf(pzsvc.TracedError("Failed to determine if clipped geometry for %v " + collection.ShoreDataID + " is empty.\n" + err.Error()).Error())
				log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			} else if empty {
				area, _ := collGeomPart.Area()
				log.Printf("Clipped geometry for %v is empty (size: %v). Continuing.", collection.ShoreDataID, area)
				log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			}
			clippedGeoms = append(clippedGeoms, clippedGeom)
		}

		if b, err = pzsvc.DownloadBytes(collection.ShoreDataID, inpObj.PzAddr, inpObj.PzAuth); err != nil {
			log.Printf(pzsvc.TracedError("Failed to download shoreline " + collection.ShoreDataID + ".\n" + err.Error()).Error())
			continue
		}

		if gjIfc, err = geojson.Parse(b); err != nil {
			log.Printf(pzsvc.TracedError("Failed to parse GeoJSON from " + collection.ShoreDataID + ".\n" + err.Error()).Error())
			continue
		}

		if fc, ok = gjIfc.(*geojson.FeatureCollection); ok {
			for _, clippedGeom = range clippedGeoms {
				if foundGeom = findBestMatch(fc, clippedGeom); foundGeom == nil {
					log.Printf("Found no matching shorelines for %v.", collection.ShoreDataID)
				} else {
					if intersectGeom, err = foundGeom.Intersection(collGeom); err != nil {
						log.Printf(pzsvc.TracedError("Failed to clip the found geometry for %v " + collection.ShoreDataID + ": " + err.Error()).Error())
						// log.Printf("foundGeom: %v", foundGeom.String())
						log.Printf("collGeom: %v", collGeom.String())
						continue
					}

					foundGeoms = append(foundGeoms, intersectGeom)
					fmt.Printf("Found a matching shoreline for %v.\n", collection.ShoreDataID)
				}
			}
		} else {
			log.Printf(pzsvc.TracedError(fmt.Sprintf("Was expecting a *geojson.FeatureCollection, got a %T", gjIfc)).Error())
		}
	}
	if result, err = geos.NewCollection(geos.GEOMETRYCOLLECTION, foundGeoms...); err != nil {
		log.Printf(pzsvc.TracedError("Failed to create new collection containing" + string(len(foundGeoms)) + " geometries\n" + err.Error()).Error())
	}
	return result
}

func findBestMatch(fc *geojson.FeatureCollection, comparison *geos.Geometry) *geos.Geometry {
	var (
		err           error
		intersects    bool
		currGeom      *geos.Geometry
		result        *geos.Geometry
		longestLength float64
	)
	log.Printf("%v features to inspect", len(fc.Features))
	for _, feature := range fc.Features {
		if currGeom, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Printf(pzsvc.TracedError("Could not convert GeoJSON object to GEOS geometry: " + err.Error()).Error())
			continue
		}
		if intersects, err = currGeom.Intersects(comparison); err != nil {
			log.Printf(pzsvc.TracedError("Failed to test intersection: " + err.Error()).Error())
			continue
		} else if intersects {
			length, _ := currGeom.Length()
			if length > longestLength {
				longestLength = length
				result = currGeom
			}
		}
	}
	return result
}
