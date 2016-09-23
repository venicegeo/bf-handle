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
	"errors"
	"strconv"

	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-lib"
)

// getMeta takes up to three sources for metadata - the S3 metadata off of a GetFileMeta
// call, the output from a call to the tide service, and one of the geojson
// features from the harvester.  It builds a map[string]string out of whichever of these
// is available and returns the result.
func getMeta(dataID, pzAddr, pzAuth string, inpTide *tideOut, feature *CatFeature) (map[string]string, error) {
	attMap := make(map[string]string)

	if dataID != "" {
		dataRes, err := pzsvc.GetFileMeta(dataID, pzAddr, pzAuth)
		if err != nil {
			return nil, err
		}

		for key, val := range dataRes.ResMeta.Metadata {
			attMap[key] = val
		}
		attMap["fileSize"] = strconv.Itoa(dataRes.DataType.Location.FileSize)
	}

	if inpTide != nil {
		attMap["24hrMinTide"] = strconv.FormatFloat(inpTide.MinTide, 'f', -1, 64)
		attMap["24hrMaxTide"] = strconv.FormatFloat(inpTide.MaxTide, 'f', -1, 64)
		attMap["currentTide"] = strconv.FormatFloat(inpTide.CurrTide, 'f', -1, 64)
	}

	if feature != nil {
		attMap["sourceID"] = feature.ID // covers source and ID in that source
		attMap["dateTimeCollect"] = feature.Properties.AcqDate
		attMap["sensorName"] = feature.Properties.SensorName
		attMap["resolution"] = strconv.Itoa(feature.Properties.Resolution)
		attMap["classification"] = feature.Properties.Classification
		if feature.Properties.Classification == "" {
			attMap["classification"] = "Unclassified"
		}
		attMap["dataUsage"] = "Not_to_be_used_for_navigational_or_targeting_purposes."
	}

	return attMap, nil
}

// addGeoFeatureMeta adds metadata to every feature in a given geojson file.  It uses the
// dataId both to download the geojson file in question from S3  It then iterates through
// all fo the features in the file and adds the given properties to each, before uploading
// the file that results and returning the dataId from that upload.
func addGeoFeatureMeta(dataID, pzAddr, pzAuth string, props map[string]string) (string, error) {
	b, err := pzsvc.DownloadBytes(dataID, pzAddr, pzAuth)
	var obj geojson.FeatureCollection
	err = json.Unmarshal(b, &obj)
	if err != nil {
		return "", errors.New("metadata.go:65: " + err.Error() + ".  input json: " + string(b))
	}

	for _, feat := range obj.Features {
		for pkey, pval := range props {
			feat.Properties[pkey] = pval
		}
	}

	b2, err := json.Marshal(obj)
	if err != nil {
		return "", errors.New("metadata.go:76" + err.Error() + ".  input json: " + string(b2))
	}

	fName := props["sourceID"] + ".geojson"
	source := props["algoName"]
	version := props["version"]

	dataID, err = pzsvc.Ingest(fName, "geojson", pzAddr, source, version, pzAuth, b2, props)

	return dataID, err
}
