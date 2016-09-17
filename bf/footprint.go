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
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/paulsmith/gogeos/geos"
	"github.com/venicegeo/geojson-geos-go/geojsongeos"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

// PrepareFootprints takes an input GeoJSON and creates a set of image features.
// Those features contain image metadata suitable for passing into a
// shoreline detection process. The geometries are footprints of the required
// regions. The output is also GeoJSON.
func PrepareFootprints(writer http.ResponseWriter, request *http.Request) {
	var (
		bytes []byte
		err   error
		gjIfc interface{}
	)

	switch request.Method {
	case "POST":
		defer request.Body.Close()
		if bytes, err = ioutil.ReadAll(request.Body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			break
		}
		if gjIfc, err = geojson.Parse(bytes); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			break
		}
		if gjIfc, err = crawlFootprints(gjIfc, nil); err == nil {
			if bytes, err = geojson.Write(gjIfc); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				break
			}
		} else {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			break
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Write(bytes)
	default:
		message := fmt.Sprintf("This endpoint does not support %v requests.", request.Method)
		http.Error(writer, message, http.StatusBadRequest)
	}
}

func crawlFootprints(gjIfc interface{}, asInpObj *asInpStruct) (*geojson.FeatureCollection, error) {
	var (
		err error
		currentGeometry,
		footprintRegion,
		captured, // The area currently covered by selected images
		points,
		point *geos.Geometry
		pointCount int
		contains   bool
		bestImage  *geojson.Feature
		bestImages *geojson.FeatureCollection
	)

	bestImages = geojson.NewFeatureCollection(nil)
	if captured, err = geos.EmptyPolygon(); err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	fmt.Print("\nProducing footprint region.")
	if footprintRegion, err = getFootprintRegion(gjIfc, 0.25); err != nil {
		return nil, err
	}
	if points, err = geojsongeos.PointCloud(footprintRegion); err != nil {
		return nil, err
	}
	if pointCount, err = points.NGeometry(); err != nil {
		return nil, pzsvc.TraceErr(err)
	}
	for inx := 0; inx < pointCount; inx++ {
		if point, err = points.Geometry(inx); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
		if contains, err = captured.Contains(point); err != nil {
			return nil, pzsvc.TraceErr(err)
		} else if contains {
			continue
		}
		if bestImage = getBestScene(point, asInpObj); bestImage == nil {
			log.Printf("Didn't get a candidate image for point %v.", point.String())
		} else {
			bestImages.Features = append(bestImages.Features, bestImage)
			if currentGeometry, err = geojsongeos.GeosFromGeoJSON(bestImage.Geometry); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
			if captured, err = captured.Union(currentGeometry); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
		}
	}
	sort.Sort(ByScore(bestImages.Features))
	fmt.Print("\nClipping footprints.")
	bestImages.Features = selfClip(bestImages.Features)
	bestImages.Features = clipFootprints(bestImages.Features, footprintRegion)

	return bestImages, nil
}

func getFootprintRegion(input interface{}, buffer float64) (*geos.Geometry, error) {
	var (
		geometries []*geos.Geometry
		geom,
		collection,
		result *geos.Geometry
		err error
	)
	switch it := input.(type) {
	case *geojson.FeatureCollection:
		for _, feature := range it.Features {
			if geom, err = getFootprintRegion(feature, buffer); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
			geometries = append(geometries, geom)
		}
		if collection, err = geos.NewCollection(geos.GEOMETRYCOLLECTION, geometries...); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
		if result, err = collection.Buffer(0); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
	case map[string]interface{}:
		return getFootprintRegion(geojson.FromMap(it), buffer)
	case *geojson.Feature, *geojson.Point, *geojson.LineString, *geojson.Polygon, *geojson.MultiPoint, *geojson.MultiLineString, *geojson.MultiPolygon, *geojson.GeometryCollection:
		if geom, err = geojsongeos.GeosFromGeoJSON(it); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
		return getFootprintRegion(geom, buffer)
	case *geos.Geometry:
		var (
			area float64
			gt   geos.GeometryType
		)
		if geom, err = it.Buffer(buffer); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
		// If we have too small a polygon, just buffer its envelope
		// so we don't waste time with a zillion points
		if gt, err = geom.Type(); err != nil {
			return nil, pzsvc.TraceErr(err)
		}
		switch gt {
		case geos.POLYGON:
			if area, err = geom.Area(); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
			if area < buffer*2.0 {
				// log.Print("Found a small geometry. Converting to Bounding Box.")
				if result, err = geom.Envelope(); err != nil {
					return nil, pzsvc.TraceErr(err)
				}
			} else {
				result = geom
				// log.Print("Found a large geometry. Simplifying.")
				// if result, err = geom.Simplify(0.1); err != nil {
				// 	return nil, pzsvc.TraceErr(err)
				// }
			}
		case geos.MULTIPOLYGON:
			var (
				count int
				geoms []*geos.Geometry
				curr  *geos.Geometry
			)
			if count, err = geom.NGeometry(); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
			for inx := 0; inx < count; inx++ {
				if curr, err = geom.Geometry(inx); err != nil {
					return nil, pzsvc.TraceErr(err)
				}
				if curr, err = getFootprintRegion(curr, 0); err != nil {
					return nil, pzsvc.TraceErr(err)
				}
				geoms = append(geoms, curr)
			}
			if result, err = geos.NewCollection(geos.MULTIPOLYGON, geoms...); err != nil {
				return nil, pzsvc.TraceErr(err)
			}
		default:
			return nil, pzsvc.TraceErr(fmt.Errorf("Unexpected geometry type: %v", gt))
		}
	default:
		return nil, pzsvc.ErrWithTrace(fmt.Sprintf("Cannot create point cloud from %T.", input))
	}
	return result, nil
}

func clipFootprints(features []*geojson.Feature, geometry *geos.Geometry) []*geojson.Feature {
	var (
		err        error
		gjGeometry interface{}
		currentGeometry,
		intersectedGeometry *geos.Geometry
		area   float64
		result []*geojson.Feature
	)

	for _, feature := range features {
		if currentGeometry, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Printf("Failed to convert GeoJSON to GEOS: %v\n%v", err.Error(), feature.String())
			continue
		}
		if intersectedGeometry, err = currentGeometry.Intersection(geometry); err != nil {
			log.Printf("Skipping current geometry: %v\n%v", currentGeometry.String(), err.Error())
			continue
		}
		if area, err = intersectedGeometry.Area(); err != nil {
			log.Printf("Failed to compute area of intersectedGeometry %v: %v", intersectedGeometry.String(), err.Error())
			continue
		}
		if area == 0.0 {
			fmt.Printf("Area of intersection for feature %v is empty. Skipping.\n", feature.ID)
			continue
		}
		if gjGeometry, err = geojsongeos.GeoJSONFromGeos(intersectedGeometry); err != nil {
			log.Printf("Failed to convert intersectedGeometry %v: %v", intersectedGeometry.String(), err.Error())
			continue
		}
		feature.Geometry = gjGeometry
		result = append(result, feature)
	}
	return result
}

func selfClip(features []*geojson.Feature) []*geojson.Feature {
	var (
		err        error
		gjGeometry interface{}
		currGeometry,
		diffGeom,
		totalGeom *geos.Geometry
		contains bool
	)
	if totalGeom, err = geos.EmptyPolygon(); err != nil {
		log.Print(err.Error())
		return features
	}
	for _, feature := range features {
		if currGeometry, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Panic(err.Error())
			return features
		}
		if contains, err = totalGeom.Contains(currGeometry); err != nil {
			log.Panic(err.Error())
			return features
		} else if !contains {
			if diffGeom, err = currGeometry.Difference(totalGeom); err != nil {
				log.Printf("totalGeometry: %v", totalGeom.String())
				log.Printf("currentGeometry: %v", currGeometry.String())
				log.Panic(err.Error())
				return features
			}
			if gjGeometry, err = geojsongeos.GeoJSONFromGeos(diffGeom); err != nil {
				log.Panic(err.Error())
				return features
			}
			feature.Geometry = gjGeometry
			if totalGeom, err = totalGeom.Union(currGeometry); err != nil {
				log.Panic(err.Error())
				return features
			}
		}
	}
	return features
}

// ByScore allows for sorting of features by their scores
type ByScore []*geojson.Feature

func (a ByScore) Len() int {
	return len(a)
}
func (a ByScore) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByScore) Less(i, j int) bool {
	return sceneScore(a[i]) < sceneScore(a[j])
}

func getBestScene(point *geos.Geometry, inpObj *asInpStruct) *geojson.Feature {
	var (
		options catalog.SearchOptions
		feature,
		currentScene,
		bestScene *geojson.Feature
		geometry interface{}
		currentScore,
		bestScore float64
		err              error
		tidesInObj       *tidesIn
		tidesOutObj      *tidesOut
		sceneDescriptors catalog.SceneDescriptors
	)
	options.NoCache = true
	options.Rigorous = true
	geometry, _ = geojsongeos.GeoJSONFromGeos(point)
	feature = geojson.NewFeature(geometry, "", nil)
	feature.Bbox = feature.ForceBbox()
	fmt.Print("\nGetting scenes.")
	if sceneDescriptors, _, err = catalog.GetScenes(feature, options); err != nil {
		log.Printf("Failed to get scenes from image catalog: %v", err.Error())
		return nil
	}
	if len(sceneDescriptors.Scenes.Features) == 0 {
		log.Printf("Found no images in catalog search. %v %#v", feature.String(), options)
		return nil
	}

	// Incorporate Tide Prediction
	if inpObj != nil && inpObj.TidesAddr != "" {
		if tidesInObj = toTidesIn(sceneDescriptors.Scenes.Features); tidesInObj != nil {
			fmt.Print("\nLoading tide information.")
			if tidesOutObj, err = getTides(*tidesInObj, inpObj.TidesAddr); err == nil {
				// Loop 1: Add the tide information to each image
				for _, tideObj := range tidesOutObj.Locations {
					currentScene = tidesInObj.Map[tideObj.Dtg]
					currentScene.Properties["CurrentTide"] = tideObj.Results.CurrTide
					currentScene.Properties["24hrMinTide"] = tideObj.Results.MinTide
					currentScene.Properties["24hrMaxTide"] = tideObj.Results.MaxTide
					updateSceneTide(currentScene, tideObj.Results)
				}
				// } else {
				// 	log.Printf("Failed to get tide prediction information: %v", err.Error())
			}
		}
	}

	// Loop 2: Check their scores
	for _, currentScene = range sceneDescriptors.Scenes.Features {
		currentScore = sceneScore(currentScene)
		if currentScore > bestScore {
			bestScene = currentScene
			bestScene.Properties["score"] = currentScore
			bestScore = currentScore
		}
	}
	return bestScene
}

var date2015 = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

func sceneScore(scene *geojson.Feature) float64 {
	var (
		result       = 1.0
		acquiredDate time.Time
		err          error
	)
	cloudCover := scene.PropertyFloat("cloudCover")
	currTide := scene.PropertyFloat("CurrentTide")
	minTide := scene.PropertyFloat("24hrMinTide")
	maxTide := scene.PropertyFloat("24hrMaxTide")
	acquiredDateString := scene.PropertyString("acquiredDate")
	if acquiredDate, err = time.Parse(time.RFC3339, acquiredDateString); err != nil {
		log.Printf("Received invalid date of %v: ", acquiredDateString)
		return 0.0
	}
	// Landsat images older than 2015 are unlikely to be in the S3 archive
	// unless they happen to have very good cloud cover so discourage them
	acquiredDateUnix := acquiredDate.Unix()
	if acquiredDateUnix < date2015 {
		result = 0.5
	}
	now := time.Now().Unix()
	result -= math.Sqrt(cloudCover / 100.0)
	result -= float64(acquiredDateUnix-now) / (60.0 * 60.0 * 24.0 * 365.0 * 10.0)
	if math.IsNaN(currTide) {
		// If no tide is available for some reason, assume low tide
		// log.Printf("No tide available for %v", scene.ID)
		result -= math.Sqrt(0.1)
	} else {
		result -= math.Sqrt(0.1) * (maxTide - currTide) / (maxTide - minTide)
	}
	return result
}

func ingestFootprints(footprints *geojson.FeatureCollection, inpObj asInpStruct) (string, []byte, error) {
	var (
		b      []byte
		result string
		err    error
	)

	// Working around annoying relational restrictions in Piazza
	footprints.FillProperties()

	// Ingest the footprints, get back the Piazza ID
	b, _ = geojson.Write(footprints)
	if result, err = pzsvc.Ingest("footprints.geojson", "geojson", inpObj.PzAddr, "bf-handle footprints", "1.0", inpObj.PzAuth, b, nil); err == nil {
		go ingestFootprintsSucceeded(result, inpObj)
	} else {
		go ingestFootprintsFailed(string(b), inpObj)
	}
	return result, b, err
}

func ingestFootprintsSucceeded(footprintsID string, inpObj asInpStruct) {
	var (
		err       error
		eventType pzsvc.EventType
	)
	etm := make(map[string]interface{})
	etm["footprintsDataID"] = "string"

	if eventType, err = pzsvc.GetEventType(":beachfront:executeBatch:footprintsIngested", etm, inpObj.PzAddr, inpObj.PzAuth); err == nil {
		event := pzsvc.Event{
			EventTypeID: eventType.EventTypeID,
			Data:        make(map[string]interface{})}
		event.Data["footprintsDataID"] = footprintsID

		if _, err = pzsvc.AddEvent(event, inpObj.PzAddr, inpObj.PzAuth); err == nil {
			fmt.Printf("Ingested footprints to Piazza, received ID %v.", footprintsID)
		} else {
			log.Printf("Failed to post event %#v\n%v", event, err.Error())
		}
	}
}
func ingestFootprintsFailed(footprints string, inpObj asInpStruct) {
	var (
		err           error
		eventType     pzsvc.EventType
		eventResponse pzsvc.Event
	)
	etm := make(map[string]interface{})
	etm["footprints"] = "string"

	if eventType, err = pzsvc.GetEventType(":beachfront:executeBatch:footprintsCalculated", etm, inpObj.PzAddr, inpObj.PzAuth); err == nil {
		event := pzsvc.Event{
			EventTypeID: eventType.EventTypeID,
			Data:        make(map[string]interface{})}
		event.Data["footprints"] = footprints

		if eventResponse, err = pzsvc.AddEvent(event, inpObj.PzAddr, inpObj.PzAuth); err == nil {
			fmt.Printf("Failed to ingest footprints to Piazza, but posted event %v.", eventResponse.EventID)
		} else {
			log.Printf("Failed to ingest footprints to Piazza or post event %#v\n%v", event, err.Error())
		}
	}
}
