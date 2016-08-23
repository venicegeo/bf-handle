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
		sourceGeometry,
		currentGeometry,
		lineString,
		polygon,
		point *geos.Geometry
		holes             []*geos.Geometry
		pointCount        int
		contains          bool
		bestImage         *geojson.Feature
		bestImages        *geojson.FeatureCollection
		currentFootprints *geojson.FeatureCollection
	)

	bestImages = geojson.NewFeatureCollection(nil)
	switch gj := gjIfc.(type) {
	case *geojson.GeometryCollection:
		for _, geometry := range gj.Geometries {
			if currentFootprints, err = crawlFootprints(geometry); err == nil {
				bestImages.Features = append(bestImages.Features, currentFootprints.Features...)
			} else {
				return nil, pzsvc.TracedError(err.Error())
			}
		}
	case *geojson.FeatureCollection:
		for _, feature := range gj.Features {
			if currentFootprints, err = crawlFootprints(feature); err == nil {
				bestImages.Features = append(bestImages.Features, currentFootprints.Features...)
			} else {
				return nil, pzsvc.TracedError(err.Error())
			}
		}
	case *geojson.MultiPolygon, *geojson.Feature, *geojson.LineString, *geojson.Polygon:
		if sourceGeometry, err = geojsongeos.GeosFromGeoJSON(gjIfc); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if sourceGeometry, err = sourceGeometry.Buffer(0.25); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if polygon, err = geos.EmptyPolygon(); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		if lineString, err = sourceGeometry.Shell(); err != nil {
			log.Printf("Shell for %v failed: %v", sourceGeometry.String(), err.Error())
			return nil, pzsvc.TracedError(err.Error())
		}
		if pointCount, err = lineString.NPoint(); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		for inx := 0; inx < pointCount; inx++ {
			if point, err = lineString.Point(inx); err != nil {
				return nil, pzsvc.TracedError(err.Error())
			}
			if contains, err = polygon.Contains(point); err != nil {
				return nil, pzsvc.TracedError(err.Error())
			} else if contains {
				continue
			}
			if bestImage = getBestImage(point); bestImage == nil {
				log.Print("Didn't get a candidate image.")
			} else {
				bestImages.Features = append(bestImages.Features, bestImage)
				if currentGeometry, err = geojsongeos.GeosFromGeoJSON(bestImage.Geometry); err != nil {
					return nil, pzsvc.TracedError(err.Error())
				}
				if polygon, err = polygon.Union(currentGeometry); err != nil {
					return nil, pzsvc.TracedError(err.Error())
				}
			}
		}
		if holes, err = sourceGeometry.Holes(); err != nil {
			return nil, pzsvc.TracedError(err.Error())
		}
		for _, hole := range holes {
			if pointCount, err = hole.NPoint(); err != nil {
				return nil, pzsvc.TracedError(err.Error())
			}
			for inx := 0; inx < pointCount; inx++ {
				if point, err = lineString.Point(inx); err != nil {
					return nil, pzsvc.TracedError(err.Error())
				}
				if contains, err = polygon.Contains(point); err != nil {
					return nil, pzsvc.TracedError(err.Error())
				} else if contains {
					continue
				}
				if bestImage = getBestImage(point); bestImage == nil {
					log.Print("Didn't get a candidate image.")
				} else {
					bestImages.Features = append(bestImages.Features, bestImage)
					if currentGeometry, err = geojsongeos.GeosFromGeoJSON(bestImage.Geometry); err != nil {
						return nil, pzsvc.TracedError(err.Error())
					}
					polygon, err = polygon.Union(currentGeometry)
				}
			}
		}
		sort.Sort(ByScore(bestImages.Features))
		bestImages.Features = selfClip(bestImages.Features)
		bestImages.Features = clipFootprints(bestImages.Features, sourceGeometry)
	default:
		return nil, pzsvc.TracedError(fmt.Sprintf("Cannot accept input of %t", gjIfc))
	}

	return bestImages, nil
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
		err                            error
		gjGeometry                     interface{}
		currentGeometry, totalGeometry *geos.Geometry
		contains                       bool
	)
	if totalGeometry, err = geos.EmptyPolygon(); err != nil {
		log.Print(err.Error())
		return features
	}
	for _, feature := range features {
		if currentGeometry, err = geojsongeos.GeosFromGeoJSON(feature); err != nil {
			log.Panic(err.Error())
			return features
		}
		// log.Print(currentGeometry.String())
		if contains, err = totalGeometry.Contains(currentGeometry); err != nil {
			log.Panic(err.Error())
			return features
		} else if !contains {
			// log.Printf("Current: %v", currentGeometry.String())
			if currentGeometry, err = currentGeometry.Difference(totalGeometry); err != nil {
				log.Panic(err.Error())
				return features
			}
			// log.Printf("Difference: %v", currentGeometry.String())
			if gjGeometry, err = geojsongeos.GeoJSONFromGeos(currentGeometry); err != nil {
				log.Panic(err.Error())
				return features
			}
			feature.Geometry = gjGeometry
			// log.Printf("GeoJSON: %v", feature.String())
			if totalGeometry, err = totalGeometry.Union(currentGeometry); err != nil {
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

func imageScore(image *geojson.Feature) float64 {
	var (
		result       float64
		acquiredDate time.Time
		err          error
	)
	cloudCover := image.PropertyFloat("cloudCover")
	acquiredDateString := image.PropertyString("acquiredDate")
	if acquiredDate, err = time.Parse(time.RFC3339, acquiredDateString); err != nil {
		log.Printf("Received invalid date of %v: ", acquiredDateString)
		return 0.0
	}
	acquiredDateUnix := acquiredDate.Unix()
	now := time.Now().Unix()
	result = 1 - (math.Sqrt(cloudCover/100.0) + (float64(now-acquiredDateUnix) / (60.0 * 60.0 * 24.0 * 365.0 * 10.0)))
	return result
}
