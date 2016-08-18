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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/montanaflynn/stats"
	"github.com/paulsmith/gogeos/geos"
)

func parseCoord(input []float64) geos.Coord {
	return geos.NewCoord(input[0], input[1])
}
func parseCoordArray(input [][]float64) []geos.Coord {
	var result []geos.Coord
	for inx := 0; inx < len(input); inx++ {
		result = append(result, parseCoord(input[inx]))
	}
	return result
}

func linearRingFromLineString(input *geos.Geometry) (*geos.Geometry, error) {
	var coords []geos.Coord
	var err error

	if coords, err = input.Coords(); err != nil {
		return nil, err
	}
	return geos.NewLinearRing(coords[:]...)
}
func lineStringFromGeometry(input *geos.Geometry) (*geos.Geometry, error) {
	var (
		coords       []geos.Coord
		result       *geos.Geometry
		err          error
		geometryType geos.GeometryType
	)

	if geometryType, err = input.Type(); err != nil {
		return result, err
	}
	switch geometryType {
	case geos.LINESTRING:
		result = input
	case geos.LINEARRING:
		coords, _ = input.Coords()
		result, _ = geos.NewLineString(coords[:]...)
	case geos.POLYGON:
		var shell *geos.Geometry
		if shell, err = input.Shell(); err != nil {
			return nil, err
		}
		// Reenter
		result, err = lineStringFromGeometry(shell)
	default:
		err = fmt.Errorf("Cannot create a line string from type %v.", geometryType)
	}
	return result, err
}

// multiPolygonize turns a slice of LineStrings into a MultiPolygon
func multiPolygonize(input []*geos.Geometry) (*geos.Geometry, error) {
	var (
		result         *geos.Geometry
		mls            *geos.Geometry
		err            error
		geometryString string
		file           *os.File
	)

	// Take the input, turn it into a MultiLineString so we can pass it to C++-land
	mls, err = geos.NewCollection(geos.MULTILINESTRING, input[:]...)
	if err != nil {
		return nil, err
	}

	// Write the MLS to a temp file as WKT
	geometryString, err = mls.ToWKT()
	file, err = ioutil.TempFile("", "mls")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())

	file.Write([]byte(geometryString))

	// Call our other application, which returns WKT
	bfla := os.Getenv("BF_LINE_ANALYZER_DIR") + "/bld/bf_la"
	cmd := exec.Command(bfla, "-mlp", file.Name())
	bytes, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	result, err = geos.FromWKT(string(bytes))

	return result, err
}

// mlsToMPoly takes a MultiLineString and turns it into a MultiPolygon
// This includes handling all of the interior (inner) rings
func mlsToMPoly(input *geos.Geometry) (*geos.Geometry, error) {
	var (
		result     *geos.Geometry
		err        error
		rings      []*geos.Geometry
		chords     []*geos.Geometry
		polygons   []*geos.Geometry
		count      int
		lineString *geos.Geometry
		ring       *geos.Geometry
		envelope   *geos.Geometry
		polygon    *geos.Geometry
		closed     bool
	)

	// Create two bins, one of rings and one of chords
	// The envelope itself is the first chord
	envelope, err = input.Envelope()
	if err != nil {
		return nil, err
	}
	ring, err = envelope.Shell()
	if err != nil {
		return nil, err
	}
	lineString, err = lineStringFromGeometry(ring)
	if err != nil {
		return nil, err
	}
	chords = append(chords, lineString)

	count, err = input.NGeometry()
	for inx := 0; inx < count; inx++ {
		lineString, err = input.Geometry(inx)
		if err != nil {
			return nil, err
		}
		closed, err = lineString.IsClosed()
		if err != nil {
			return nil, err
		}
		if closed {
			ring, err = linearRingFromLineString(lineString)
			if err != nil {
				return nil, err
			}
			rings = append(rings, ring)
		} else {
			chords = append(chords, lineString)
		}
	}

	// Create a MultiPolygon covering the AOI
	if len(chords) > 1 {
		result, err = multiPolygonize(chords)
	} else {
		result, err = geos.NewCollection(geos.MULTIPOLYGON, envelope)
	}
	if err != nil {
		return nil, err
	}

	// Make a new bag of polygons,
	// associating the detected rings with the right polygon
	count, err = result.NGeometry()
	if err != nil {
		return nil, err
	}
	for inx := 0; inx < count; inx++ {
		var (
			innerRings []*geos.Geometry
			contains   bool
		)
		polygon, err = result.Geometry(inx)
		if err != nil {
			return nil, err
		}
		for jnx := 0; jnx < len(rings); jnx++ {
			contains, err = polygon.Contains(rings[jnx])
			if err != nil {
				return nil, err
			}
			if contains {
				innerRings = append(innerRings, rings[jnx])
			}
		}
		ring, err = polygon.Shell()
		if err != nil {
			return nil, err
		}
		polygon, err = geos.PolygonFromGeom(ring, innerRings[:]...)
		if err != nil {
			return nil, err
		}
		polygons = append(polygons, polygon)
	}

	// Reconstruct the MultiPolygon
	result, err = geos.NewCollection(geos.MULTIPOLYGON, polygons[:]...)

	return result, err
}

func lineStringsToFloat64Data(first, second *geos.Geometry) (stats.Float64Data, error) {
	var (
		err      error
		coords   []geos.Coord
		data     []float64
		distance float64
		point    *geos.Geometry
	)

	if first, err = lineStringFromGeometry(first); err != nil {
		return nil, err
	}
	coords, _ = first.Coords()
	data = make([]float64, len(coords))
	for inx := range coords {
		if point, err = geos.NewPoint(coords[inx]); err != nil {
			return nil, err
		}
		if distance, err = point.Distance(second); err != nil {
			return nil, err
		}
		data[inx] = distance
	}
	return stats.LoadRawData(data), err
}

func centroidCoordsXY(input *geos.Geometry) (float64, float64, error) {
	var (
		centroid         *geos.Geometry
		resultX, resultY float64
		err              error
	)
	if centroid, err = input.Centroid(); err != nil {
		return 0, 0, err
	}
	if resultX, err = centroid.X(); err != nil {
		return 0, 0, err
	}
	if resultY, err = centroid.Y(); err != nil {
		return 0, 0, err
	}
	return resultX, resultY, err
}

func displace(input *geos.Geometry, xShift float64, yShift float64) (*geos.Geometry, error) {
	var (
		coords []geos.Coord
		err    error
	)
	if coords, err = input.Coords(); err != nil {
		return nil, err
	}
	for inx := range coords {
		coords[inx].X -= xShift
		coords[inx].Y -= yShift
	}
	return geos.NewLineString(coords[:]...)
}
