/*
Copyright 2016, RadiantBlue Technologies, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package analyze

import (
	"log"
	"testing"

	"github.com/paulsmith/gogeos/geos"
	"github.com/venicegeo/geojson-go/geojson"
)

// TestScene Unit test for this object
func TestScene(t *testing.T) {
	var (
		envelope      *geos.Geometry
		baselineScene Scene
		detectedScene Scene
		err           error
	)
	filenameB := "test/baseline.geojson"
	filenameD := "test/detected.geojson"
	if baselineScene.geoJSON, err = geojson.ParseFile(filenameB); err != nil {
		t.Errorf("Failed to parse input file %v: %v", filenameB, err.Error())
	}
	if detectedScene.geoJSON, err = geojson.ParseFile(filenameD); err != nil {
		t.Errorf("Failed to parse input file %v: %v", filenameD, err.Error())
	}
	if envelope, err = detectedScene.envelope(); err != nil {
		t.Errorf("Failed to produced the detected scene envelope: %v", err.Error())
	}
	log.Printf("Envelope: %v\n", envelope.String())
	if err = baselineScene.clip(detectedScene); err != nil {
		t.Error(err.Error())
	}
	if envelope, err = baselineScene.envelope(); err != nil {
		t.Error(err.Error())
	}
	log.Printf("Envelope: %v\n", envelope.String())
}

// TestDisplace tries out displacing a Geos Geometry
func TestDisplace(t *testing.T) {
	coords := [...]geos.Coord{{X: 0.0, Y: 1.0}, {X: 2.0, Y: 2.0}}
	geom, _ := geos.NewLineString(coords[:]...)
	log.Printf("Geom: %v", geom.String())
	geom, _ = displace(geom, 2, 3)
	log.Printf("Geom: %v", geom.String())
}
