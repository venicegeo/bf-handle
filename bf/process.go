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
	
	"github.com/venicegeo/pzsvc-lib"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
)

/*
Various TODOs:
- put in at least a few more comments
- improve error handling (currently *very* rudimentary)
*/

type inpStruct struct {
	AlgoType	string			`json:"algoType"`
	AlgoURL		string			`json:"svcURL"`
	BndMrgType	string			`json:"bandMergeType"`
	BndMrgURL	string			`json:"bandMergeURL"`
	MetaJSON	geojson.Feature	`json:"metaDataJSON"`
	Bands		[]string		`json:"bands"`
	PzAuth		string			`json:"pzAuthToken"`
	PzAddr		string			`json:"pzAddr"`
	DbAuth		string			`json:"dbAuthToken"`
}

type outpStruct struct {
	ShoreDataID	string			`json:"shoreDataID"`
	RGBloc		string			`json:"rgbLoc"`
	Error		string			`json:"error"`
}

// GenShoreline serves as main function for this file, and is the
// primary workhorse function of bf-handle as a whole.  It
// processes raster images into geojson.
func GenShoreline (w http.ResponseWriter, r *http.Request) {
	var inpObj inpStruct
	var outpObj outpStruct
	var rgbChan chan string

	// handleOut is a subfunction for making sure that the output is
	// handled in a consistent manner.
	handleOut := func (errmsg string, status int) {
		outpObj.Error = errmsg
		b, err := json.Marshal(outpObj)
		if err != nil {
			fmt.Fprintf(w, `{"error":"json.Marshal error: `+err.Error()+`", "baseError":"`+errmsg+`"}`)
		}
		http.Error(w, string(b), status)
		return
	}

	fmt.Println ("bf-handle called.")		
	inpBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handleOut("Error: ioutil.ReadAll: " + err.Error(), http.StatusBadRequest)
		return
	}	
	
	err = json.Unmarshal(inpBytes, &inpObj)
	if err != nil {
		handleOut("Error: json.Unmarshal: " + err.Error(), http.StatusBadRequest)
		return
	}

	(&inpObj.MetaJSON).ResolveGeometry()
	
	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}
	
	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	if inpObj.BndMrgType != "" && inpObj.BndMrgURL != "" {
		rgbChan = make(chan string)
		go rgbGen(inpObj, rgbChan)
	}

	fmt.Println ("bf-handle: provisioning begins.")
	dataIDs, err := provision(inpObj, nil)
	if err != nil{
		handleOut("Error: bf-handle provisioning: " + err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println ("bf-handle: running Algo")
	outpObj.ShoreDataID, err = runAlgo(inpObj, dataIDs)
	if err != nil{
		handleOut("Error: algo result: " + err.Error(), http.StatusBadRequest)
		return
	}
	if rgbChan != nil {
		fmt.Println ("waiting for rgb")
		rgbLoc := <-rgbChan
		if len(rgbLoc) > 7 && rgbLoc[0:6] == "Error:" {
			handleOut(rgbLoc, http.StatusInternalServerError)
			return
		}
		outpObj.RGBloc = rgbLoc
	}

	fmt.Println ("outputting")
	handleOut("", http.StatusOK)
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.

func provision(inpObj inpStruct, bands []string) ( []string, error ) {
	
	if bands == nil {
		bands = inpObj.Bands
	}
	dataIDs := make([]string, len(bands))
	fSource := inpObj.MetaJSON.PropertyString("sensorName")

	for i, band := range bands {
fmt.Println ("provisioning: Beginning " + band + " band.")
		reader, err := catalog.ImageFeatureIOReader(&inpObj.MetaJSON, band, inpObj.DbAuth)
		if err != nil {
			return nil, fmt.Errorf(`catalog.ImageFeatureIOReader: %s`, err.Error())
		}
		fName := fmt.Sprintf("%s-%s.TIF", fSource, band)

		bSlice, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf(`ioutil.Readall: %s`, err.Error())
		}
fmt.Println ("provisioning: Bytes acquired.  Beginning ingest.")
		// TODO: at some point, we might wish to add properties to the TIFF files as we ingest them.
		// We'd do that by replacing the "nil", below, with an appropriate map[string]string.
		dataID, err := pzsvc.Ingest(fName, "raster", inpObj.PzAddr, fSource, "", inpObj.PzAuth, bSlice, nil)
		if err != nil {
			return nil, fmt.Errorf(`pzsvc.Ingest: %s`, err.Error())
		}
		dataIDs[i] = dataID
fmt.Println ("provisioning: Ingest completed.")
	}
	return dataIDs, nil
}

// 
func runAlgo( inpObj inpStruct, dataIDs []string) (string, error) {
	var dataID string
	var attMap map[string]string
	var err error
	hasFeatMeta := false
	switch inpObj.AlgoType {
	case "pzsvc-ossim":
		attMap, err = getMeta("","","",&inpObj.MetaJSON)
		if err != nil {
			return "", fmt.Errorf(`getMeta: %s`, err.Error())
		}
		dataID, err = runOssim (inpObj.AlgoURL, dataIDs[0], dataIDs[1], inpObj.PzAuth, attMap)
		if err != nil {
			return "", fmt.Errorf(`runOssim: %s`, err.Error())
		}
//		hasFeatMeta = true  // Currently, Ossim does not have feature-level metadata after all.
							// until that's fixed, we need to treat them teh same way we do
							// everyone else.
	default:
		return "", fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, inpObj.AlgoType)
	}

	attMap, err = getMeta (dataID, inpObj.PzAddr, inpObj.PzAuth, &inpObj.MetaJSON)
	if err != nil{
		return "", fmt.Errorf(`getMeta2: %s`, err.Error())
	}

	if hasFeatMeta {
		return dataID, pzsvc.UpdateFileMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
	}
	
	return addGeoFeatureMeta(dataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
}

// runOssim does all of the things necessary to process the given images
// through pzsvc-ossim.  It constructs and executes the request, reads
// the response, and extracts the dataID of the output from it.
func runOssim(algoURL, imgID1, imgID2, authKey string, attMap map[string]string ) (string, error) {
	geoJName := `shoreline.geojson`

	funcStr := fmt.Sprintf(`shoreline --image %s.TIF,%s.TIF --projection geo-scaled `, imgID1, imgID2)
	for key, val := range attMap {
		funcStr = funcStr + fmt.Sprintf(`--prop %s:%s `, key, val)
	}
	funcStr = funcStr + geoJName

	inpObj := pzsvc.ExecIn{	FuncStr:funcStr,
							InFiles:[]string{0:imgID1,1:imgID2},
							OutGeoJSON:[]string{0:geoJName},
							OutGeoTIFF:nil,
							OutTxt:nil,
							AlgoURL:algoURL,
							AuthKey:authKey}


	outMap, err := pzsvc.CallPzsvcExec(&inpObj)
	if err != nil {
		return "", fmt.Errorf(`CallPzsvcExec: %s`, err.Error())
	}
	return outMap[geoJName], nil
}