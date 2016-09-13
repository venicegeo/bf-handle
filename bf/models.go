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
	"github.com/venicegeo/geojson-go/geojson"
)

// CatProp reresents the properties if the pzsvc-image-catalog
// feature objects.  It's exported to make sure that it plays
// well with json unmarshaling.
type CatProp struct {
	AcqDate        string            `json:"acquiredDate"`
	Bands          map[string]string `json:"bands"`
	CacheDataID    string            `json:"cache.shoreDataID"`
	CacheDeplID    string            `json:"cache.shoreDeplID"`
	CloudCover     float64           `json:"cloudCover"`
	FileFormat     string            `json:"fileFormat"`
	Classification string            `json:"classification"`
	Path           string            `json:"path"`
	Resolution     int               `json:"resolution"`
	SensorName     string            `json:"sensorName"`
	LgThumb        string            `json:"thumb_large"`
	SmThumb        string            `json:"thumb_small"`
}

// CatFeature is the format for the Feature objects from pzsvc-image-catalog.
// If we could we'd just use the geojson.Feature type, but in this case, we
// need to guide the json unmarshaling process a bit too directly for that.
// Having the parts we care about fully and explicitly defined also makes it
// easier to dig down for specific bits of information, when that's called for.
// Exported to make sure it plays well with json unmarshaling
type CatFeature struct {
	Type       string              `json:"type"`
	Geometry   interface{}         `json:"geometry"`
	Properties CatProp             `json:"properties"`
	ID         string              `json:"id"`
	BBox       geojson.BoundingBox `json:"bbox"`
}
