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
	"os"
	"runtime/debug"

	"github.com/paulsmith/gogeos/geos"
	"github.com/venicegeo/geojson-geos-go/geojsongeos"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

type asInpStruct struct {
	AlgoType string `json:"algoType"` // API for the shoreline algorithm
	AlgoURL  string `json:"svcURL"`   // URL for the shoreline algorithm
	// BndMrgType string           `json:"bandMergeType,omitempty"` // API for the bandmerge/rgb algorithm (optional)
	// BndMrgURL  string           `json:"bandMergeURL,omitempty"`  // URL for the bandmerge/rgb algorithm (optional)
	Bands            []string                   `json:"bands"`                 // names of bands to feed into the shoreline algorithm
	PzAuth           string                     `json:"pzAuthToken,omitempty"` // Auth string for this Pz instance
	PzAddr           string                     `json:"pzAddr"`                // gateway URL for this Pz instance
	DbAuth           string                     `json:"dbAuthToken,omitempty"` // Auth string for the initial image database
	LGroupID         string                     `json:"lGroupId"`              // UUID string for the target geoserver layer group
	JobName          string                     `json:"resultName"`            // Arbitrary user-defined string to aid in later reference
	TidesAddr        string                     `json:"tidesAddr"`             // URL for Tide Prediction Service (optional)
	Collections      *geojson.FeatureCollection `json:"collections"`           // Collection objects
	Baseline         map[string]interface{}     `json:"baseline"`              // Baseline shoreline, as GeoJSON
	FootprintsDataID string                     `json:"footprintsDataID"`      // Piazza ID of GeoJSON containing footprints
	SkipDetection    bool                       `json:"skipDetection"`         // true: skip detection; go straight to assembly
	ForceDetection   bool                       `json:"forceDetection"`        // true: ignore cache
}

type ebOutStruct struct {
	FootprintsDataID string           `json:"footprintsDataID"` // Piazza ID for GeoJSON of footprints
	FootprintsDepl   *pzsvc.DeplStrct `json:"footprintsDepl"`   // Piazza ID for GeoJSON of footprints
	ShoreDataID      string           `json:"shoreDataID"`      // Piazza ID for GeoJSON of shorelines
	ShoreDepl        *pzsvc.DeplStrct `json:"shoreDepl"`        // Piazza ID for GeoJSON of shorelines
}

// AssembleShorelines creates a single dataset from some input or something
func AssembleShorelines(w http.ResponseWriter, r *http.Request) {
	var (
		b          []byte
		err        error
		inpObj     asInpStruct
		outpObj    gsOutpStruct
		shorelines *geojson.FeatureCollection
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
		errStr := pzsvc.TraceStr("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(b))
		handleError(errStr, http.StatusBadRequest)
		return
	}

	if shorelines, err = assembleShorelines(inpObj); err != nil {
		handleError(err.Error(), http.StatusBadRequest)
	} else {
		if b, err = geojson.Write(shorelines); err != nil {
			errStr := pzsvc.TraceStr("Failed to write output GeoJSON object: " + err.Error())
			handleError(errStr, http.StatusInternalServerError)
			return
		}
		w.Write(b)
	}
}

// ExecuteBatch executes a single shoreline detection
// based on a GeoJSON object representing one or more geometries
func ExecuteBatch(w http.ResponseWriter, r *http.Request) {
	var (
		b          []byte
		err        error
		inpObj     asInpStruct
		shorelines *geojson.FeatureCollection
		gen        *geojson.Feature
		footprints *geojson.FeatureCollection
		result     ebOutStruct
		gsInpObj   gsInpStruct
		shoreDataID,
		shoreDeplID string
	)

	// clients to this function expect a JSON response
	// containing the error message
	handleError := func(errmsg string, status int) {
		log.Print(errmsg)
		outpErr := pzsvc.Error{Message: errmsg}
		b, err = json.Marshal(outpErr)
		if err != nil {
			b = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		http.Error(w, string(b), status)
	}

	if b, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		handleError(pzsvc.TraceStr("Error: pzsvc.ReadBodyJSON: "+err.Error()+".\nInput String: "+string(b)), http.StatusBadRequest)
		return
	}

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}
	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	if inpObj.FootprintsDataID == "" {
		if inpObj.Baseline == nil {
			handleError(pzsvc.TraceStr("Input must contain a baseline FeatureCollection or a FootprintsDataID."), http.StatusBadRequest)
			return
		}

		if footprints, err = crawlFootprints(inpObj.Baseline, &inpObj); err != nil {
			handleError(pzsvc.TraceStr("Error: failed to crawl footprints: "+err.Error()), http.StatusInternalServerError)
			return
		}

		// Ingest the footprints, store the Piazza ID
		if result.FootprintsDataID, b, err = ingestFootprints(footprints, inpObj); err == nil {
			if result.FootprintsDepl, err = pzsvc.DeployToGeoServer(result.FootprintsDataID, "", inpObj.PzAddr, inpObj.PzAuth); err == nil {
				fmt.Printf("Deployed footprints go GeoServer. DeplID: %v", result.FootprintsDepl.DeplID)
			} else {
				log.Printf(pzsvc.TraceStr("Failed to deploy footprint GeoJSON to GeoServer: " + err.Error()))
			}
		}
	} else {
		if b, err = pzsvc.DownloadBytes(inpObj.FootprintsDataID, inpObj.PzAddr, inpObj.PzAuth); err == nil {
			if footprints, err = geojson.FeatureCollectionFromBytes(b); err != nil {
				errStr := pzsvc.TraceStr("Error: Failed to build FeatureCollection from contents of ID " + inpObj.FootprintsDataID + ": " + err.Error())
				handleError(errStr, http.StatusBadRequest)
				return
			}
			// The footprints information is abbreviated and might not contain information
			// on previous collection operations so re-retrieve from the catalog
			var newFootprint *geojson.Feature
			for inx, footprint := range footprints.Features {
				if newFootprint, err = catalog.GetImageMetadata(footprint.ID); err == nil {
					footprints.Features[inx] = newFootprint
				} else {
					log.Printf("Failed to retrieve image %v from catalog.", footprint.ID)
				}
			}
			result.FootprintsDataID = inpObj.FootprintsDataID
		} else {
			errStr := pzsvc.TraceStr("Error: Failed to download footprints from ID " + inpObj.FootprintsDataID + ": " + err.Error())
			handleError(errStr, http.StatusBadRequest)
			return
		}
	}

	if len(footprints.Features) == 0 {
		handleError(pzsvc.TraceStr("No footprint features in input."), http.StatusBadRequest)
		return
	}

	// Convert the asInpStruct to a gsInpStruct
	b, _ = json.Marshal(inpObj)
	json.Unmarshal(b, &gsInpObj)

	fmt.Printf("\nReady to start shoreline assembly. Input object: %#v", gsInpObj)
	inpObj.Collections = geojson.NewFeatureCollection(nil)

	for inx, footprint := range footprints.Features {
		if shoreDataID = footprint.PropertyString("cache.shoreDataID"); inpObj.ForceDetection || shoreDataID == "" {
			if !inpObj.SkipDetection {
				fmt.Printf("Detecting scene %v (#%v of %v, score %v)\n", footprint.ID, inx+1, len(footprints.Features), sceneScore(footprint))

				if gen, err = popShoreline(gsInpObj, footprint); err != nil {
					log.Printf("Failed to detect scene %v: %v", footprint.ID, err.Error())
					continue
				}
				inpObj.Collections.Features = append(inpObj.Collections.Features, gen)
				shoreDataID = gen.PropertyString("shoreDataID")
				shoreDeplID = gen.PropertyString("shoreDeplID")
				fmt.Printf("Finished detecting feature %v. Data ID: %v\n", footprint.ID, shoreDataID)
				go addCache(footprint.ID, shoreDataID, shoreDeplID)
				debug.FreeOSMemory()
			}
		} else {
			fmt.Printf("Found Data ID %v for feature %v\n", shoreDataID, footprint.ID)
			footprint.Properties["shoreDataID"] = shoreDataID
			footprint.Properties["shoreDeplID"] = shoreDeplID
			inpObj.Collections.Features = append(inpObj.Collections.Features, footprint)
		}
	}

	fmt.Print("\nFinished shoreline generation. Starting assembly.")

	if shorelines, err = assembleShorelines(inpObj); err != nil {
		handleError(err.Error(), http.StatusInternalServerError)
		return
	}

	// Ingest the shorelines, store the Piazza ID in outpObj
	ingest := true

	// Working around annoying relational restrictions in Piazza
	shorelines.FillProperties()

	b, _ = geojson.Write(shorelines)
	if result.ShoreDataID, err = pzsvc.Ingest("shorelines.geojson", "geojson", inpObj.PzAddr, "bf-handle ExecuteBatch", "1.0", inpObj.PzAuth, b, nil); err == nil {
		if result.ShoreDepl, err = pzsvc.DeployToGeoServer(result.ShoreDataID, "", inpObj.PzAddr, inpObj.PzAuth); err != nil {
			ingest = false
			log.Printf(pzsvc.TraceStr("Failed to deploy shorelines GeoJSON to GeoServer: " + err.Error()))
		}
	} else {
		ingest = false
		log.Printf(pzsvc.TraceStr("Failed to ingest shorelines GeoJSON: " + err.Error()))
		for inx := 0; inx < 100 && inx < len(shorelines.Features); inx++ {
			log.Printf("%#v", shorelines.Features[inx].Properties)
		}
	}

	// If the ingest works, writes the output object
	// If not, just write the detected shorelines JSON
	if ingest {
		b, _ = json.Marshal(result)
		log.Printf("Completed batch process: \n%v", string(b))
	}
	w.Write(b)
	w.Header().Set("Content-Type", "application/json")
}

func assembleShorelines(inpObj asInpStruct) (*geojson.FeatureCollection, error) {
	var (
		gjIfc interface{}
		baseline,
		collGeom,
		collGeomPart,
		clippedGeom *geos.Geometry
		b   []byte
		err error
		currFc,
		fc *geojson.FeatureCollection
		ok     bool
		empty  bool
		result *geojson.FeatureCollection
		foundGeoms,
		clippedGeoms []*geos.Geometry
		count       int
		shoreDataID string
	)
	if baseline, err = geojsongeos.GeosFromGeoJSON(inpObj.Baseline); err != nil {
		return nil, pzsvc.ErrWithTrace("Could not convert GeoJSON object to GEOS geometry: " + err.Error())
	}

	result = geojson.NewFeatureCollection(nil)

	for _, collection := range inpObj.Collections.Features {
		clippedGeoms = nil
		foundGeoms = nil
		b = nil
		debug.FreeOSMemory()

		shoreDataID = collection.PropertyString("shoreDataID")
		if collGeom, err = geojsongeos.GeosFromGeoJSON(collection.Geometry); err != nil {
			log.Printf("%T", collection.Geometry)
			log.Printf(pzsvc.TraceStr("Could not convert GeoJSON object to GEOS geometry: " + err.Error()))
			continue
		}

		// Because this can't be easy, the intersection function doesn't work well with multipolygons.
		// We have to split them apart and test them individually
		if count, err = collGeom.NGeometry(); err != nil {
			log.Printf(pzsvc.TraceStr("Could not count the parts of the collected geometry: " + err.Error()))
			log.Printf("collGeom: %v", collGeom.String())
			continue
		}

		for inx := 0; inx < count; inx++ {
			collGeomPart, _ = collGeom.Geometry(inx)

			if clippedGeom, err = baseline.Intersection(collGeomPart); err != nil {
				log.Printf(pzsvc.TraceStr("Could not clip the baseline geometry: " + err.Error()))
				log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			}

			if empty, err = clippedGeom.IsEmpty(); err != nil {
				log.Printf(pzsvc.TraceStr("Failed to determine if clipped geometry for %v " + shoreDataID + " is empty.\n" + err.Error()))
				log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			} else if empty {
				area, _ := collGeomPart.Area()
				log.Printf("Clipped geometry for %v is empty (size: %v). Continuing.", shoreDataID, area)
				// log.Printf("collGeomPart: %v", collGeomPart.String())
				continue
			}
			clippedGeoms = append(clippedGeoms, clippedGeom)
		}

		if b, err = pzsvc.DownloadBytes(shoreDataID, inpObj.PzAddr, inpObj.PzAuth); err != nil {
			log.Printf(pzsvc.TraceStr("Failed to download shoreline " + shoreDataID + ".\n" + err.Error()))
			continue
		}

		if gjIfc, err = geojson.Parse(b); err != nil {
			log.Printf(pzsvc.TraceStr("Failed to parse GeoJSON from " + shoreDataID + ".\n" + err.Error()))
			continue
		}

		b = nil
		debug.FreeOSMemory()

		if fc, ok = gjIfc.(*geojson.FeatureCollection); ok {
			for _, clippedGeom = range clippedGeoms {
				if currFc = findBestMatches(fc, clippedGeom, collGeom); len(currFc.Features) == 0 {
					log.Printf("Found no matching shorelines for %v.", shoreDataID)
				} else {
					result.Features = append(result.Features, currFc.Features...)
					fmt.Printf("Found %v matching shorelines for %v.\n", len(currFc.Features), shoreDataID)
				}
			}
			debug.FreeOSMemory()
		} else {
			log.Printf(pzsvc.TraceStr(fmt.Sprintf("Was expecting a *geojson.FeatureCollection, got a %T", gjIfc)))
		}
		if gjIfc, err = geos.NewCollection(geos.GEOMETRYCOLLECTION, foundGeoms...); err != nil {
			log.Printf(pzsvc.TraceStr("Failed to create new collection containing" + string(len(foundGeoms)) + " geometries\n" + err.Error()))
		}
	}
	return result, nil
}

func findBestMatches(fc *geojson.FeatureCollection, comparison, clip *geos.Geometry) *geojson.FeatureCollection {
	var (
		err         error
		intersects  bool
		gjIfc       interface{}
		currFeature *geojson.Feature
		currGeom,
		intersectGeom *geos.Geometry
		result *geojson.FeatureCollection
	)
	result = geojson.NewFeatureCollection(nil)
	fmt.Printf("%v features to inspect. ", len(fc.Features))
	for _, feature := range fc.Features {
		if currGeom, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Printf(pzsvc.TraceStr("Could not convert GeoJSON object to GEOS geometry: " + err.Error()))
			continue
		}
		// Need a better test here?
		if intersects, err = currGeom.Intersects(comparison); err != nil {
			log.Printf(pzsvc.TraceStr("Failed to test intersection: " + err.Error()))
			continue
		} else if intersects {
			// Need to clip each found geometry to its collection geometry
			if intersectGeom, err = currGeom.Intersection(clip); err != nil {
				log.Printf(pzsvc.TraceStr("Failed to clip the found geometry for %v " + feature.ID + ": " + err.Error()))
				// log.Printf("clip: %v", clip.String())
				// log.Printf("currGeom: %v", currGeom.String())
				continue
			}

			if gjIfc, err = geojsongeos.GeoJSONFromGeos(intersectGeom); err != nil {
				log.Printf(pzsvc.TraceStr("Failed to convert GEOS geometry to GeoJSON: " + err.Error()))
				log.Printf("intersectGeom: %v", intersectGeom.String())
				continue
			}
			currFeature = geojson.NewFeature(gjIfc, feature.ID, feature.Properties)
			result.Features = append(result.Features, currFeature)
		}
	}
	return result
}

func addCache(imageID, shoreDataID, shoreDeplID string) {
	var (
		feature *geojson.Feature
		err     error
	)
	// Get a clean copy of the image metadata
	if feature, err = catalog.GetImageMetadata(imageID); err != nil {
		log.Printf(pzsvc.TraceStr("Failed to retrieve image metadata so that we could cache results: " + err.Error()))
	}

	feature.Properties["cache.shoreDataID"] = shoreDataID
	feature.Properties["cache.shoreDeplID"] = shoreDeplID

	// re-store the feature
	if _, err = catalog.StoreFeature(feature, true); err != nil {
		log.Printf(pzsvc.TraceStr("Failed to store image metadata with cached results: " + err.Error()))
	}
}
