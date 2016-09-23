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
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/venicegeo/geojson-go/geojson"
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
	BndMrgType string           `json:"bandMergeType,omitempty"` // API for the bandmerge/rgb service (optional)
	BndMrgURL  string           `json:"bandMergeURL,omitempty"`  // URL for the bandmerge/rgb service (optional)
	TideURL    string           `json:"tideURL,omitempty"`       // URL for the tide service (optional)
	MetaJSON   *CatFeature      `json:"metaDataJSON,omitempty"`  // JSON block from Image Catalog
	MetaURL    string           `json:"metaDataURL,omitempty"`   // URL to call to get JSON block
	metaFeat   *geojson.Feature ``                               // in place to maintain support with bulk-builds
	Bands      []string         `json:"bands"`                   // names of bands to feed into the shoreline algorithm
	PzAuth     string           `json:"pzAuthToken,omitempty"`   // Auth string for this Pz instance
	PzAddr     string           `json:"pzAddr"`                  // gateway URL for this Pz instance
	DbAuth     string           `json:"dbAuthToken,omitempty"`   // Auth string for the initial image database
	LGroupID   string           `json:"lGroupId"`                // UUID string for the target geoserver layer group
	JobName    string           `json:"jobName"`                 // Arbitrary user-defined string to aid in later reference
}

type gsOutpStruct struct {
	ShoreDataID  string      `json:"shoreDataID"`
	ShoreDeplID  string      `json:"shoreDeplID"`
	RGBloc       string      `json:"rgbLoc"`
	Geometry     interface{} `json:"geometry"`
	AlgoType     string      `json:"algoType"`
	SceneCapDate string      `json:"sceneCaptureDate"`
	SceneID      string      `json:"sceneId"`
	JobName      string      `json:"resultName"`
	SensorName   string      `json:"sensorName"`
	AlgoURL      string      `json:"svcURL"`
	Error        string      `json:"error"`
}

// Execute executes a single shoreline detection
// based on the metadata in a gsInpStruct
func Execute(w http.ResponseWriter, r *http.Request) {
	var (
		byts        []byte
		err         error
		inpObj      gsInpStruct
		outpObj     gsOutpStruct
		outpFeature *genShoreOut
	)

	// clients to this function expect a JSON response
	// containing the error message
	handleError := func(errmsg string, status int) {
		outpObj.Error = errmsg
		byts, err = json.Marshal(outpObj)
		if err != nil {
			byts = []byte(`{"error":"json.Marshal error: ` + err.Error() + `", "baseError":"` + errmsg + `"}`)
		}
		pzsvc.HTTPOut(w, string(byts), status)
	}

	if byts, err = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		errStr := pzsvc.TraceStr("Error: pzsvc.ReadBodyJSON: " + err.Error() + ".\nInput String: " + string(byts))
		handleError(errStr, http.StatusBadRequest)
		return
	}

	if (inpObj.MetaURL == "") == (inpObj.MetaJSON == nil) {
		errStr := pzsvc.TraceStr("Error: Must specify one and only one of metaDataURL (" + inpObj.MetaURL + ") and metaDataJSON.")
		handleError(errStr, http.StatusBadRequest)
		return
	}

	if inpObj.MetaURL != "" {
		inpObj.MetaJSON = new(CatFeature)
		if _, err = pzsvc.RequestKnownJSON("GET", "", inpObj.MetaURL, inpObj.PzAuth, inpObj.MetaJSON); err != nil {
			errStr := pzsvc.TraceStr("Error: pzsvc.RequestKnownJSON: possible flaw in metaDataURL (" + inpObj.MetaURL + "): " + err.Error())
			handleError(errStr, http.StatusBadRequest)
			return
		}
	}

	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}

	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	if outpFeature, err = genShoreline(inpObj); err == nil {
		outpObj.JobName = inpObj.JobName
		outpObj.AlgoType = inpObj.AlgoType
		outpObj.SceneID = inpObj.MetaJSON.ID
		outpObj.SceneCapDate = inpObj.MetaJSON.Properties.AcqDate
		outpObj.Geometry = inpObj.MetaJSON.Geometry
		outpObj.SensorName = inpObj.MetaJSON.Properties.SensorName
		outpObj.AlgoURL = inpObj.AlgoURL
		outpObj.ShoreDataID = outpFeature.dataID
		outpObj.ShoreDeplID = outpFeature.deplID
		outpObj.RGBloc = outpFeature.rgbLoc

		w.Header().Set("Content-Type", "application/json")
		byts, _ = json.Marshal(outpObj)
		w.Write(byts)
	} else {
		handleError(err.Error(), http.StatusInternalServerError)
	}
}

type genShoreOut struct {
	minTide  float64
	maxTide  float64
	currTide float64
	dataID   string
	deplID   string
	rgbLoc   string
}

// popShoreline functions serves as an in to genShoreline for
// those who want to get a geojson.Feature out.
func popShoreline(inpObj gsInpStruct, inFeat *geojson.Feature) (*geojson.Feature, error) {
	var (
		byts     []byte
		err      error
		shoreOut *genShoreOut
	)

	if inFeat == nil {
		return nil, pzsvc.ErrWithTrace("Error: Must specify Feature.")
	}

	if byts, err = json.Marshal(inFeat); err != nil {
		return nil, pzsvc.TraceErr(err)
	}

	if err = json.Unmarshal(byts, &inpObj.MetaJSON); err != nil {
		return nil, pzsvc.TraceErr(err)
	}

	shoreOut, err = genShoreline(inpObj)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}

	inFeat.Properties["24hrMinTide"] = strconv.FormatFloat(shoreOut.minTide, 'f', -1, 64)
	inFeat.Properties["24hrMaxTide"] = strconv.FormatFloat(shoreOut.maxTide, 'f', -1, 64)
	inFeat.Properties["currentTide"] = strconv.FormatFloat(shoreOut.currTide, 'f', -1, 64)
	inFeat.Properties["shoreDataID"] = shoreOut.dataID
	inFeat.Properties["shoreDeplID"] = shoreOut.deplID

	return inFeat, nil

}

// genShoreline serves as main function for this file, and is the
// primary workhorse function of bf-handle as a whole.  It
// processes raster images into geojson.
func genShoreline(inpObj gsInpStruct) (*genShoreOut, error) {
	var (
		result      genShoreOut
		rgbChan     chan string
		err         error
		urls        []string
		shoreDataID string
		deplObj     *pzsvc.DeplStrct
		inTideObj   *tideIn
		outTideObj  *tideOut
	)

	if inpObj.BndMrgType != "" && inpObj.BndMrgURL != "" {
		rgbChan = make(chan string)
		go rgbGen(inpObj, rgbChan)
	}

	if inpObj.TideURL != "" {
		if inTideObj = findTide(inpObj.MetaJSON.BBox, inpObj.MetaJSON.Properties.AcqDate); inTideObj == nil {
			return nil, pzsvc.TraceErr(
				fmt.Errorf(`Could not get tide information from feature %v because 
					required elements did not exist.`, inpObj.MetaJSON.ID))
		}

		// currently, the tide prediction service can generate
		// error-producing output even with valid requests (for
		// example, if the scene is in the middle of the ocean).
		// Thus, if we get an error from this, we simply continue
		// without the tide data.
		if outTideObj, err = getTide(*inTideObj, inpObj.TideURL); err == nil {
			result.minTide = outTideObj.MinTide
			result.maxTide = outTideObj.MaxTide
			result.currTide = outTideObj.CurrTide
		} else {
			fmt.Printf("Skipping tide information for %v: %v", inpObj.MetaJSON.ID, err.Error())
		}
	}

	if urls, err = findImgURLs(inpObj); err != nil {
		return &result, pzsvc.TraceErr(err)
	}

	fmt.Println("bf-handle: running Algo")
	if shoreDataID, deplObj, err = runAlgo(inpObj, outTideObj, urls); err != nil {
		return &result, pzsvc.TraceErr(err)
	}
	result.dataID = shoreDataID
	result.deplID = deplObj.DeplID

	if rgbChan != nil {
		fmt.Println("waiting for rgb")
		rgbLoc := <-rgbChan // returns the Geoserver Layer
		if len(rgbLoc) > 7 && rgbLoc[0:6] == "Error:" {
			return &result, errors.New(rgbLoc)
		}
		result.rgbLoc = rgbLoc
	}

	return &result, nil
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.
/*
func provision(inpObj gsInpStruct, bands []string) ([]string, error) {

	if bands == nil {
		bands = inpObj.Bands
	}
	dataIDs := make([]string, len(bands))
	fSource := inpObj.MetaJSON.Properties.SensorName

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
}*/

func findImgURLs(inpObj gsInpStruct) ([]string, error) {
	outURLs := make([]string, len(inpObj.Bands))
	for i, band := range inpObj.Bands {
		outURLs[i] = inpObj.MetaJSON.Properties.Bands[band]
	}
	return outURLs, nil
}

// runAlgo does whatever it takes to run the algorithm it is given on
// the dataIDs it is told to target.  It returns the dataId of the result
// file.  Right now, it doesn't have any algorithms to handle other than
// pzsvc-ossim, but as that changes the case statement is going to get
// bigger and uglier.
func runAlgo(inpObj gsInpStruct, inpTide *tideOut, inpURLs []string) (string, *pzsvc.DeplStrct, error) {
	var (
		dataID      string
		attMap      map[string]string
		deplObj     *pzsvc.DeplStrct
		err         error
		hasFeatMeta = false
	)
	switch inpObj.AlgoType {
	case "pzsvc-ossim":
		attMap, err = getMeta("", "", "", inpTide, inpObj.MetaJSON)
		if err != nil {
			return "", nil, fmt.Errorf(`getMeta: %s`, err.Error())
		}
		dataID, err = runOssim(inpObj.AlgoURL, inpURLs[0], inpURLs[1], inpObj.PzAuth, attMap)
		if err != nil {
			return "", nil, fmt.Errorf(`runOssim: %s`, err.Error())
		}
		//		hasFeatMeta = true
		// the version of OSSIM we are currently capable of using does not have feature-level
		// metadata.  Until/unless that's fixed, we need to treat them the same way we do
		// everyone else.
	default:
		return "", nil, fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, inpObj.AlgoType)
	}

	attMap, err = getMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, inpTide, inpObj.MetaJSON)
	if err != nil {
		return "", nil, fmt.Errorf(`getMeta2: %s`, err.Error())
	}

	if hasFeatMeta {
		err = pzsvc.UpdateFileMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
		if err != nil {
			return "", nil, fmt.Errorf(`pzsvc.UpdateFileMeta: %s`, err.Error())
		}
	} else {
		dataID, err = addGeoFeatureMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
		if err != nil {
			return "", nil, fmt.Errorf(`addGeoFeatureMeta: %s`, err.Error())
		}
	}

	deplObj, err = pzsvc.DeployToGeoServer(dataID, inpObj.LGroupID, inpObj.PzAddr, inpObj.PzAuth)
	if err != nil {
		return "", nil, fmt.Errorf(`pzsvc.DeployToGeoServer: %s`, err.Error())
	}

	fmt.Printf("Completed algorithm %v; %v : %v", inpObj.MetaJSON.ID, dataID, deplObj.DeplID)

	return dataID, deplObj, nil
}

// runOssim does all of the things necessary to process the given images
// through pzsvc-ossim.  It constructs and executes the request, reads
// the response, and extracts the dataID of the output from it.
func runOssim(algoURL, imgURL1, imgURL2, authKey string, attMap map[string]string) (string, error) {
	geoJName := `shoreline.geojson`

	funcStr := `shoreline --image img1.TIF,img2.TIF --projection geo-scaled `
	for key, val := range attMap {
		funcStr = funcStr + fmt.Sprintf(`--prop %s:%s `, key, val)
	}
	funcStr = funcStr + geoJName

	inpObj := pzsvc.ExecIn{FuncStr: funcStr,
		InExtURLs:  []string{0: imgURL1, 1: imgURL2},
		InExtNames: []string{0: "img1.TIF", 1: "img2.TIF"},
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
