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
	"io/ioutil"
	"net/http"
	"os"

	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
)

/*
Various TODOs:
- put in at least a few more comments
- improve error handling (currently *very* rudimentary)
*/

type gsInpStruct struct {
	AlgoType   string           `json:"algoType"`                // API for the shoreline algorithm
	AlgoURL    string           `json:"svcURL"`                  // URL for the shoreline algorithm
	BndMrgType string           `json:"bandMergeType,omitempty"` // API for the bandmerge/rgb algorithm (optional)
	BndMrgURL  string           `json:"bandMergeURL,omitempty"`  // URL for the bandmerge/rgb algorithm (optional)
	MetaJSON   *geojson.Feature `json:"metaDataJSON,omitempty"`  // JSON block from Image Catalog
	MetaURL    string           `json:"metaDataURL,omitempty"`   // URL to call to get JSON block
	Bands      []string         `json:"bands"`                   // names of bands to feed into the shoreline algorithm
	PzAuth     string           `json:"pzAuthToken,omitempty"`   // Auth string for this Pz instance
	PzAddr     string           `json:"pzAddr"`                  // gateway URL for this Pz instance
	DbAuth     string           `json:"dbAuthToken,omitempty"`   // Auth string for the initial image database
	LGroupID   string           `json:"lGroupId"`                // UUID string for the target geoserver layer group
	JobName    string           `json:"resultName"`              // Arbitrary user-defined string to aid in later reference
}

type gsOutpStruct struct {
	ShoreDataID string      `json:"shoreDataID"`
	ShoreDeplID string      `json:"shoreDeplID"`
	RGBloc      string      `json:"rgbLoc"`
	Geometry    interface{} `json:"geometry"`
	AlgoType    string      `json:"algorithmName"`
	ImgCapDate  string      `json:"imageCaptureDate"`
	DbImgID     string      `json:"dbImageId"`
	JobName     string      `json:"resultName"`
	SensorName  string      `json:"sensorName"`
	AlgoURL     string      `json:"svcURL"`
	Error       string      `json:"error"`
}

type ebOutpStruct struct {
	Collections []gsOutpStruct `json:"collections"`
	Error       string         `json:"error"`
}

// Execute executes a single shoreline detection
// based on the metadata in a gsInpStruct
func Execute(w http.ResponseWriter, r *http.Request) {
	var (
		b       []byte
		err     error
		inpObj  gsInpStruct
		outpObj gsOutpStruct
	)

	// clients to this function expect a JSON response
	// containing the error message
	handleError := func(errmsg string, status int) {
		outpObj.Error = errmsg
		b, err = json.Marshal(outpObj)
		if err != nil {
			b = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		pzsvc.HTTPOut(w, string(b), status)
	}

	if b, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		tracedError := pzsvc.TracedError("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(b))
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	if (inpObj.MetaURL == "") == (inpObj.MetaJSON == nil) {
		tracedError := pzsvc.TracedError("Error: Must specify one and only one of metaDataURL (" + inpObj.MetaURL + ") and metaDataJSON.")
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	if inpObj.MetaURL != "" {
		inpObj.MetaJSON = geojson.NewFeature(nil, "", nil)
		if _, err = pzsvc.RequestKnownJSON("GET", "", inpObj.MetaURL, inpObj.PzAuth, inpObj.MetaJSON); err != nil {
			tracedError := pzsvc.TracedError("Error: pzsvc.RequestKnownJSON: possible flaw in metaDataURL (" + inpObj.MetaURL + "): " + err.Error())
			handleError(tracedError.Error(), http.StatusBadRequest)
			return
		}
	}

	inpObj.MetaJSON.ResolveGeometry()

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	outpObj = genShoreline(inpObj)
	if outpObj.Error == "" {
		w.Header().Set("Content-Type", "application/json")
		b, _ = json.Marshal(outpObj)
		w.Write(b)
	} else {
		handleError(outpObj.Error, http.StatusInternalServerError)
	}
}

// ExecuteBatch executes a single shoreline detection
// based on a GeoJSON object representing one or more geometries
func ExecuteBatch(w http.ResponseWriter, r *http.Request) {
	var (
		b       []byte
		err     error
		inpObj  gsInpStruct
		outpObj ebOutpStruct
	)

	// clients to this function expect a JSON response
	// containing the error message
	handleError := func(errmsg string, status int) {
		outpObj.Error = errmsg
		b, err = json.Marshal(outpObj)
		if err != nil {
			b = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		http.Error(w, string(b), status)
	}

	if b, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		tracedError := pzsvc.TracedError("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(b))
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	// In this case the metadataJSON points to a feature containing the geometries to crawl over
	if (inpObj.MetaURL == "") == (inpObj.MetaJSON == nil) {
		tracedError := pzsvc.TracedError(`Error: Must specify one and only one of metaDataURL ("` + inpObj.MetaURL + `") and metaDataJSON.`)
		handleError(tracedError.Error(), http.StatusBadRequest)
		return
	}

	if inpObj.MetaURL != "" {
		if _, err = pzsvc.RequestKnownJSON("GET", "", inpObj.MetaURL, inpObj.PzAuth, inpObj.MetaJSON); err != nil {
			tracedError := pzsvc.TracedError("Error: pzsvc.RequestKnownJSON: possible flaw in metaDataURL (" + inpObj.MetaURL + "): " + err.Error())
			handleError(tracedError.Error(), http.StatusBadRequest)
			return
		}
	}

	inpObj.MetaJSON.ResolveGeometry()

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	outpObj = genShorelineBatch(inpObj)
	fmt.Print("Finished batch process.")

	if outpObj.Error == "" {
		w.Header().Set("Content-Type", "application/json")
		b, _ = json.Marshal(outpObj)
		fmt.Print(string(b))
		w.Write(b)
	} else {
		handleError(outpObj.Error, http.StatusInternalServerError)
	}
}

func genShorelineBatch(inpObj gsInpStruct) ebOutpStruct {
	var (
		result ebOutpStruct
		gen    gsOutpStruct
		fc     *geojson.FeatureCollection
		err    error
	)

	if fc, err = crawlFootprints(inpObj.MetaJSON); err != nil {
		result.Error = "Error: failed to crawl footprints: " + err.Error()
		return result
	}

	if len(fc.Features) == 0 {
		result.Error = "No footprint features in input."
		return result
	}

	for _, footprint := range fc.Features {
		fmt.Printf("Collecting feature %v", footprint.ID)
		inpObj.MetaJSON = footprint
		gen = genShoreline(inpObj)
		if gen.Error != "" {
			result.Error = fmt.Sprintf("%v\nReceived error %v on ID %v", result.Error, inpObj.MetaJSON.ID, gen.Error)
		}
		result.Collections = append(result.Collections, gen)
	}

	if result.Error != "" {
		result.Error = "Failures were detected in shoreline detections:" + result.Error
	}
	return result
}

// GenShoreline serves as main function for this file, and is the
// primary workhorse function of bf-handle as a whole.  It
// processes raster images into geojson.
func genShoreline(inpObj gsInpStruct) gsOutpStruct {
	var (
		outpObj gsOutpStruct
		rgbChan chan string
		err     error
		dataIDs []string
	)

	if inpObj.BndMrgType != "" && inpObj.BndMrgURL != "" {
		rgbChan = make(chan string)
		go rgbGen(inpObj, rgbChan)
	}

	outpObj.JobName = inpObj.JobName
	outpObj.AlgoType = inpObj.AlgoType
	outpObj.DbImgID = inpObj.MetaJSON.ID
	outpObj.ImgCapDate = inpObj.MetaJSON.Properties["acquiredDate"].(string)
	outpObj.Geometry = inpObj.MetaJSON.Geometry
	outpObj.SensorName = inpObj.MetaJSON.Properties["sensorName"].(string)
	outpObj.AlgoURL = inpObj.AlgoURL

	fmt.Println("bf-handle: running provision")
	if dataIDs, err = provision(inpObj, nil); err != nil {
		outpObj.Error = err.Error()
		return outpObj
	}

	fmt.Println("bf-handle: running Algo")
	outpObj.ShoreDataID, outpObj.ShoreDeplID, err = runAlgo(inpObj, dataIDs)
	if err != nil {
		outpObj.Error = err.Error()
		return outpObj
	}

	if rgbChan != nil {
		fmt.Println("waiting for rgb")
		rgbLoc := <-rgbChan
		if len(rgbLoc) > 7 && rgbLoc[0:6] == "Error:" {
			outpObj.Error = rgbLoc
			return outpObj
		}
		outpObj.RGBloc = rgbLoc
	}

	fmt.Println("outputting")
	return outpObj
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.
func provision(inpObj gsInpStruct, bands []string) ([]string, error) {

	if bands == nil {
		bands = inpObj.Bands
	}
	dataIDs := make([]string, len(bands))
	fSource := inpObj.MetaJSON.PropertyString("sensorName")

	for i, band := range bands {
		fmt.Println("provisioning: Beginning " + band + " band.")
		reader, err := catalog.ImageFeatureIOReader(inpObj.MetaJSON, band, inpObj.DbAuth)
		if err != nil {
			return nil, fmt.Errorf(`catalog.ImageFeatureIOReader: %s`, err.Error())
		}
		fName := fmt.Sprintf("%s-%s.TIF", fSource, band)

		bSlice, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf(`ioutil.Readall: %s`, err.Error())
		}
		fmt.Println("provisioning: Bytes acquired.  Beginning ingest.")
		// TODO: at some point, we might wish to add properties to the TIFF files as we ingest them.
		// We'd do that by replacing the "nil", below, with an appropriate map[string]string.
		dataID, err := pzsvc.Ingest(fName, "raster", inpObj.PzAddr, fSource, "", inpObj.PzAuth, bSlice, nil)
		if err != nil {
			return nil, fmt.Errorf(`pzsvc.Ingest: %s`, err.Error())
		}
		dataIDs[i] = dataID
		fmt.Println("provisioning: Ingest completed.")
	}
	return dataIDs, nil
}

// runAlgo does whatever it takes to run the algorithm it is given on
// the dataIDs it is told to target.  It returns the dataId of the result
// file.  Right now, it doesn't have any algorithms to handle other than
// pzsvc-ossim, but as that changes the case statement is going to get
// bigger and uglier.
func runAlgo(inpObj gsInpStruct, dataIDs []string) (string, string, error) {
	var dataID, deplID string
	var attMap map[string]string
	var err error
	hasFeatMeta := false
	switch inpObj.AlgoType {
	case "pzsvc-ossim":
		attMap, err = getMeta("", "", "", inpObj.MetaJSON)
		if err != nil {
			return "", "", fmt.Errorf(`getMeta: %s`, err.Error())
		}
		dataID, err = runOssim(inpObj.AlgoURL, dataIDs[0], dataIDs[1], inpObj.PzAuth, attMap)
		if err != nil {
			return "", "", fmt.Errorf(`runOssim: %s`, err.Error())
		}
		//		hasFeatMeta = true  // Currently, Ossim does not have feature-level metadata after all.
		// until/unless that's fixed, we need to treat them the same way we do
		// everyone else.
	default:
		return "", "", fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, inpObj.AlgoType)
	}

	attMap, err = getMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, inpObj.MetaJSON)
	if err != nil {
		return "", "", fmt.Errorf(`getMeta2: %s`, err.Error())
	}

	if hasFeatMeta {
		err = pzsvc.UpdateFileMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
		if err != nil {
			return "", "", fmt.Errorf(`pzsvc.UpdateFileMeta: %s`, err.Error())
		}
	} else {
		dataID, err = addGeoFeatureMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
		if err != nil {
			return "", "", fmt.Errorf(`addGeoFeatureMeta: %s`, err.Error())
		}
	}

	deplID, err = pzsvc.DeployToGeoServer(dataID, inpObj.LGroupID, inpObj.PzAddr, inpObj.PzAuth)
	if err != nil {
		return "", "", fmt.Errorf(`pzsvc.DeployToGeoServer: %s`, err.Error())
	}

	return dataID, deplID, nil
}

// runOssim does all of the things necessary to process the given images
// through pzsvc-ossim.  It constructs and executes the request, reads
// the response, and extracts the dataID of the output from it.
func runOssim(algoURL, imgID1, imgID2, authKey string, attMap map[string]string) (string, error) {
	geoJName := `shoreline.geojson`

	funcStr := fmt.Sprintf(`shoreline --image %s.TIF,%s.TIF --projection geo-scaled `, imgID1, imgID2)
	for key, val := range attMap {
		funcStr = funcStr + fmt.Sprintf(`--prop %s:%s `, key, val)
	}
	funcStr = funcStr + geoJName

	inpObj := pzsvc.ExecIn{FuncStr: funcStr,
		InFiles:    []string{0: imgID1, 1: imgID2},
		OutGeoJSON: []string{0: geoJName},
		OutGeoTIFF: nil,
		OutTxt:     nil,
		AlgoURL:    algoURL,
		AuthKey:    authKey}

	outStruct, err := pzsvc.CallPzsvcExec(&inpObj)
	if err != nil {
		return "", fmt.Errorf(`CallPzsvcExec: %s`, err.Error())
	}
	return outStruct.OutFiles[geoJName], nil
}
