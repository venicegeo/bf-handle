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
	case pzsvc.MethodPost:
		defer request.Body.Close()
		if bytes, err = ioutil.ReadAll(request.Body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			break
		}
		if gjIfc, err = geojson.Parse(bytes); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			break
		}
		if gjIfc, err = crawlFootprints(gjIfc); err == nil {
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

func crawlFootprints(gjIfc interface{}) (*geojson.FeatureCollection, error) {
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
		return nil, pzsvc.TracedError(err.Error())
	}
	if footprintRegion, err = getFootprintRegion(gjIfc, 0.25); err != nil {
		return nil, err
	}
	if points, err = geojsongeos.PointCloud(footprintRegion); err != nil {
		return nil, err
	}
	if pointCount, err = points.NGeometry(); err != nil {
		return nil, pzsvc.TracedError(err.Error())
	}
	for inx := 0; inx < pointCount; inx++ {
		if point, err = points.Geometry(inx); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if contains, err = captured.Contains(point); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		} else if contains {
			continue
		}
		if bestImage = getBestImage(point); bestImage == nil {
			log.Printf("Didn't get a candidate image for point %v.", point.String())
		} else {
			bestImages.Features = append(bestImages.Features, bestImage)
			if currentGeometry, err = geojsongeos.GeosFromGeoJSON(bestImage.Geometry); err != nil {
				return nil, pzsvc.TracedError(err.Error())
			}
			if captured, err = captured.Union(currentGeometry); err != nil {
				return nil, pzsvc.TracedError(err.Error())
			}
		}
	}
	sort.Sort(ByScore(bestImages.Features))
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
				return nil, pzsvc.TracedError(err.Error())
			}
			geometries = append(geometries, geom)
		}
		if collection, err = geos.NewCollection(geos.GEOMETRYCOLLECTION, geometries...); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if result, err = collection.Buffer(0); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
	case map[string]interface{}:
		return getFootprintRegion(geojson.FromMap(it), buffer)
	case *geojson.Feature, *geojson.Point, *geojson.LineString, *geojson.Polygon, *geojson.MultiPoint, *geojson.MultiLineString, *geojson.MultiPolygon, *geojson.GeometryCollection:
		if geom, err = geojsongeos.GeosFromGeoJSON(it); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if result, err = geom.Buffer(buffer); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
	default:
		return nil, pzsvc.TracedError(fmt.Sprintf("Cannot create point cloud from %T.", input))
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
	return imageScore(a[i]) < imageScore(a[j])
}

func getBestImage(point *geos.Geometry) *geojson.Feature {
	var (
		options catalog.SearchOptions
		feature,
		currentImage,
		bestImage *geojson.Feature
		geometry interface{}
		currentScore,
		bestScore float64
		err              error
		imageDescriptors catalog.ImageDescriptors
	)
	options.NoCache = true
	options.Rigorous = true
	geometry, _ = geojsongeos.GeoJSONFromGeos(point)
	feature = geojson.NewFeature(geometry, "", nil)
	feature.Bbox = feature.ForceBbox()
	if imageDescriptors, _, err = catalog.GetImages(feature, options); err != nil {
		log.Printf("Failed to get images from image catalog: %v", err.Error())
		return nil
	}
	if len(imageDescriptors.Images.Features) == 0 {
		log.Printf("Found no images in catalog search.")
	}
	for _, currentImage = range imageDescriptors.Images.Features {
		currentScore = imageScore(currentImage)
		if currentScore > bestScore {
			bestImage = currentImage
			bestImage.Properties["score"] = currentScore
			bestScore = currentScore
		}
	}
	return bestImage
}

var date2015 = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

func imageScore(image *geojson.Feature) float64 {
	var (
		result       float64
		acquiredDate time.Time
		err          error
		baseline     = 1.0
	)
	cloudCover := image.PropertyFloat("cloudCover")
	acquiredDateString := image.PropertyString("acquiredDate")
	if acquiredDate, err = time.Parse(time.RFC3339, acquiredDateString); err != nil {
		log.Printf("Received invalid date of %v: ", acquiredDateString)
		return 0.0
	}
	// Landsat images older than 2015 are unlikely to be in the S3 archive
	// unless they happen to have very good cloud cover so discourage them
	acquiredDateUnix := acquiredDate.Unix()
	if acquiredDateUnix < date2015 {
		baseline = 0.5
	}
	now := time.Now().Unix()

	result = baseline - (math.Sqrt(cloudCover/100.0) + (float64(now-acquiredDateUnix) / (60.0 * 60.0 * 24.0 * 365.0 * 10.0)))
	return result
}
