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
	Bands       []string                   `json:"bands"`                 // names of bands to feed into the shoreline algorithm
	PzAuth      string                     `json:"pzAuthToken,omitempty"` // Auth string for this Pz instance
	PzAddr      string                     `json:"pzAddr"`                // gateway URL for this Pz instance
	DbAuth      string                     `json:"dbAuthToken,omitempty"` // Auth string for the initial image database
	LGroupID    string                     `json:"lGroupId"`              // UUID string for the target geoserver layer group
	JobName     string                     `json:"resultName"`            // Arbitrary user-defined string to aid in later reference
	Collections *geojson.FeatureCollection `json:"collections"`           // Collection objects
	Baseline    map[string]interface{}     `json:"baseline"`              // Baseline shoreline, as GeoJSON
}

type ebOutStruct struct {
	FootprintsID string `json:"footprintsID"` // Piazza ID for GeoJSON of footprints
	ShorelinesID string `json:"shorelinesID"` // Piazza ID for GeoJSON of shorelines
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
		tracedError := pzsvc.TracedError("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(b))
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	if shorelines, err = assembleShorelines(inpObj); err != nil {
		handleError(err.Error(), http.StatusBadRequest)
	} else {
		if b, err = geojson.Write(shorelines); err != nil {
			tracedError := pzsvc.TracedError("Failed to write output GeoJSON object: " + err.Error())
			handleError(tracedError.Error(), http.StatusInternalServerError)
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
		outpErr := pzsvc.Error{Message: errmsg}
		b, err = json.Marshal(outpErr)
		if err != nil {
			b = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		http.Error(w, string(b), status)
	}

	if b, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		handleError(pzsvc.TracedError("Error: pzsvc.ReadBodyJSON: "+err.Error()+".\nInput String: "+string(b)).Error(), http.StatusBadRequest)
		return
	}

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	if footprints, err = crawlFootprints(inpObj.Baseline); err != nil {
		handleError(pzsvc.TracedError("Error: failed to crawl footprints: "+err.Error()).Error(), http.StatusInternalServerError)
		return
	}

	if len(footprints.Features) == 0 {
		handleError(pzsvc.TracedError("No footprint features in input.").Error(), http.StatusInternalServerError)
		return
	}

	// Ingest the footprints, store the Piazza ID in outpObj
	if result.FootprintsID, b, err = writeFootprints(footprints, inpObj); err != nil {
		log.Printf(pzsvc.TracedError("Failed to ingest footprint GeoJSON: " + err.Error()).Error())
		log.Print(string(b))
	}

	b, _ = json.Marshal(inpObj)
	json.Unmarshal(b, &gsInpObj)

	log.Printf("Input object: %#v", gsInpObj)
	inpObj.Collections = geojson.NewFeatureCollection(nil)

	for inx, footprint := range footprints.Features {
		if shoreDataID = footprint.PropertyString("cache.shoreDataID"); shoreDataID == "" {
			fmt.Printf("Collecting scene %v (#%v of %v)\n", footprint.ID, inx, len(footprints.Features))
			gsInpObj.MetaJSON = footprint
			if gen, _, err = genShoreline(gsInpObj); err != nil {
				log.Printf("Failed to collect feature %v: %v", footprint.ID, err.Error())
				continue
			}
			inpObj.Collections.Features = append(inpObj.Collections.Features, gen)
			shoreDataID = gen.PropertyString("shoreDataID")
			shoreDeplID = gen.PropertyString("shoreDeplID")
			fmt.Printf("Finished collecting feature %v. Data ID: %v\n", footprint.ID, shoreDataID)
			go addCache(footprint.ID, shoreDataID, shoreDeplID)
		} else {
			fmt.Printf("Found Data ID %v for feature %v\n", shoreDataID, footprint.ID)
			footprint.Properties["shoreDataID"] = shoreDataID
			footprint.Properties["shoreDeplID"] = shoreDeplID
			inpObj.Collections.Features = append(inpObj.Collections.Features, footprint)
		}
	}

	log.Print("Finished shoreline generation. Starting assembly.")

	if shorelines, err = assembleShorelines(inpObj); err != nil {
		handleError(err.Error(), http.StatusInternalServerError)
		return
	}

	// Ingest the shorelines, store the Piazza ID in outpObj
	b, _ = geojson.Write(shorelines)
	if result.ShorelinesID, err = pzsvc.Ingest("shorelines.geojson", "geojson", inpObj.PzAddr, inpObj.AlgoType, "1.0", inpObj.PzAuth, b, nil); err == nil {
		b, _ = json.Marshal(result)
	} else {
		log.Printf(pzsvc.TracedError("Failed to ingest shorelines GeoJSON: " + err.Error()).Error())
	}

	w.Write(b)
	w.Header().Set("Content-Type", "application/json")
}

func writeFootprints(footprints *geojson.FeatureCollection, inpObj asInpStruct) (string, []byte, error) {
	var (
		writeFc    = geojson.NewFeatureCollection(nil)
		feature    *geojson.Feature
		properties map[string]interface{}
		b          []byte
		result     string
		err        error
	)

	// We can't write some properties to Piazza
	for _, footprint := range footprints.Features {
		properties = make(map[string]interface{})
		for key, property := range footprint.Properties {
			switch key {
			case "bands", "cache.shoreDataID", "cache.shoreDeplID":
				continue
			default:
				properties[key] = property
			}
		}
		feature = geojson.NewFeature(footprint.Geometry, footprint.ID, properties)
		writeFc.Features = append(writeFc.Features, feature)
	}

	// Ingest the footprints, store the Piazza ID in outpObj
	b, _ = geojson.Write(writeFc)
	result, err = pzsvc.Ingest("footprints.geojson", "geojson", inpObj.PzAddr, inpObj.AlgoType, "1.0", inpObj.PzAuth, b, nil)
	return result, b, err
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
		return nil, pzsvc.TracedError("Could not convert GeoJSON object to GEOS geometry: " + err.Error())
	}

	result = geojson.NewFeatureCollection(nil)

	for _, collection := range inpObj.Collections.Features {
		clippedGeoms = nil
		foundGeoms = nil
		shoreDataID = collection.PropertyString("shoreDataID")
		if collGeom, err = geojsongeos.GeosFromGeoJSON(collection.Geometry); err != nil {
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
				log.Printf(pzsvc.TracedError("Failed to determine if clipped geometry for %v " + shoreDataID + " is empty.\n" + err.Error()).Error())
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
			log.Printf(pzsvc.TracedError("Failed to download shoreline " + shoreDataID + ".\n" + err.Error()).Error())
			continue
		}

		if gjIfc, err = geojson.Parse(b); err != nil {
			log.Printf(pzsvc.TracedError("Failed to parse GeoJSON from " + shoreDataID + ".\n" + err.Error()).Error())
			continue
		}

		if fc, ok = gjIfc.(*geojson.FeatureCollection); ok {
			for _, clippedGeom = range clippedGeoms {
				if currFc = findBestMatches(fc, clippedGeom, collGeom); len(currFc.Features) == 0 {
					log.Printf("Found no matching shorelines for %v.", shoreDataID)
				} else {
					result.Features = append(result.Features, currFc.Features...)
					fmt.Printf("Found %v matching shorelines for %v.\n", len(currFc.Features), shoreDataID)
				}
			}
		} else {
			log.Printf(pzsvc.TracedError(fmt.Sprintf("Was expecting a *geojson.FeatureCollection, got a %T", gjIfc)).Error())
		}
		if gjIfc, err = geos.NewCollection(geos.GEOMETRYCOLLECTION, foundGeoms...); err != nil {
			log.Printf(pzsvc.TracedError("Failed to create new collection containing" + string(len(foundGeoms)) + " geometries\n" + err.Error()).Error())
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
	log.Printf("%v features to inspect", len(fc.Features))
	for _, feature := range fc.Features {
		if currGeom, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Printf(pzsvc.TracedError("Could not convert GeoJSON object to GEOS geometry: " + err.Error()).Error())
			continue
		}
		// Need a better test here
		if intersects, err = currGeom.Intersects(comparison); err != nil {
			log.Printf(pzsvc.TracedError("Failed to test intersection: " + err.Error()).Error())
			continue
		} else if intersects {
			// Need to clip each found geometry to its collection geometry
			if intersectGeom, err = currGeom.Intersection(clip); err != nil {
				log.Printf(pzsvc.TracedError("Failed to clip the found geometry for %v " + feature.ID + ": " + err.Error()).Error())
				// log.Printf("clip: %v", clip.String())
				// log.Printf("currGeom: %v", currGeom.String())
				continue
			}

			if gjIfc, err = geojsongeos.GeoJSONFromGeos(intersectGeom); err != nil {
				log.Printf(pzsvc.TracedError("Failed to convert GEOS geometry to GeoJSON: " + err.Error()).Error())
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
		log.Printf(pzsvc.TracedError("Failed to retrieve image metadata so that we could cache results: " + err.Error()).Error())
	}

	feature.Properties["cache.shoreDataID"] = shoreDataID
	feature.Properties["cache.shoreDeplID"] = shoreDeplID

	// re-store the feature
	if _, err = catalog.StoreFeature(feature, true); err != nil {
		log.Printf(pzsvc.TracedError("Failed to store image metadata with cached results: " + err.Error()).Error())
	}
}
