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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/geojson-go/geojson"
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
)

/*
Various TODOs:
- clean/refactor
- put in at least a few more comments
- improve error handling (currently *very* rudimentary)
*/

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		}
		// Stop here if its Preflighted OPTIONS request
		if r.Method == "OPTIONS" {
			return
		}

		r.ParseForm()
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, "hello.")
		case "/execute":
			proc (w,r)
		default:
			fmt.Fprintf(w, "Command undefined.  Try help?\n")
		}
	})
	
	portStr := ":8085"
	portEnv := os.Getenv("PORT")
	if portEnv != "" {
		portStr = fmt.Sprintf(":%s", portEnv)
	}	
	log.Fatal(http.ListenAndServe(portStr, nil))
}

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

func proc (w http.ResponseWriter, r *http.Request) {
	var inpObj inpStruct
	var outpObj outpStruct

	handleOut := func (errmsg string, status int) {
		outpObj.Error = errmsg
		w.WriteHeader(status)
		b, err := json.Marshal(outpObj)
		if err != nil {
			fmt.Fprintf(w, `{"error":"json.Marshal error: `+err.Error()+`", "baseError":"`+errmsg+`"}`)
		}
		fmt.Fprintf(w, string(b))
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

	fmt.Println ("bf-handle: provisioning begins.")
	dataIDs, err := provision(inpObj)
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

	fmt.Println ("outputting")
	handleOut("", http.StatusOK)
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.

func provision(inpObj inpStruct) ( []string, error ) {
	
	dataIDs := make([]string, len(inpObj.Bands))

	fSource := inpObj.MetaJSON.PropertyString("sensorName")

	for i, band := range inpObj.Bands {
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
			return "", err
		}
		dataID, err = runOssim (inpObj.AlgoURL, dataIDs[0], dataIDs[1], inpObj.PzAuth, attMap)
		if err != nil {
			return "", err
		}
		hasFeatMeta = true
	default:
		return "", fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, inpObj.AlgoType)
	}

	attMap, err = getMeta (dataID, inpObj.PzAddr, inpObj.PzAuth, &inpObj.MetaJSON)
	if err != nil{
		return "", err
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
	type execStruct struct {
		InFiles		map[string]string
		OutFiles	map[string]string
		ProgReturn	string
		Errors		[]string
	}
	
	geoJName := `shoreline.geojson`

	funcStr := fmt.Sprintf(`shoreline -i %s.TIF,%s.TIF `, imgID1, imgID2)
	for key, val := range attMap {
		funcStr = funcStr + fmt.Sprintf(`--prop %s:%s `, key, val)
	}
	funcStr = funcStr + geoJName

	//funcStr := fmt.Sprintf(`shoreline --image %s.TIF,%s.TIF --projection geo-scaled --threshold 0.5 --tolerance 0 %s`, imgID1, imgID2, geoJName)

	inStr := fmt.Sprintf(`%s,%s`, imgID1, imgID2)

	var formVal url.Values
	formVal = make(map[string][]string)
	formVal.Set("cmd", funcStr)
	formVal.Set("inFiles", inStr)
	formVal.Set("outGeoJson", geoJName)
	formVal.Set("authKey", authKey)
	fmt.Println(funcStr)
	fmt.Println(inStr)
	fmt.Println(geoJName)
	resp, err := http.PostForm(algoURL, formVal)
	if err != nil {
		return "", err
	}
	
	respBuf := &bytes.Buffer{}
	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	var respObj execStruct
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}
	
	outDataID := respObj.OutFiles[geoJName]
	if outDataID == "" {
		errstr := `Error: could not find outfile.  Likely failure in pzsvc-ossim call.`
		return "", fmt.Errorf("%s  JSON output: %s", errstr, respBuf.String())
	}
	
	return outDataID, nil

}

// getMeta takes up to three sources for metadata - the S3 metadata off of a GetFileMeta
// call, an existing map[string]string, and the useful parts of one fo the geojson
// features from the harvester.  It builds a map[string]string out of whichever of these
// is available and returns the result.
func getMeta(dataID, pzAddr, pzAuth string, feature *geojson.Feature) (map[string]string, error) {
	attMap := make(map[string]string)

	if dataID != "" {
		dataRes, err := pzsvc.GetFileMeta(dataID, pzAddr, pzAuth)
		if err != nil {
			return nil, err
		}

		for key, val := range dataRes.Metadata.Metadata {
			attMap[key] = val
		}
	}
	
	if feature != nil {
		attMap["sourceID"] = feature.ID // covers source and ID in that source
		attMap["dateTimeCollect"] = feature.PropertyString("acquiredDate")
		attMap["sensorName"] = feature.PropertyString("sensorName")
		attMap["resolution"] = feature.PropertyString("resolution")
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
		return "", err
	}

	for _, feat := range obj.Features{
		for pkey, pval := range props{
			feat.Properties[pkey] = pval
		}
	}

	b2, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	fName := props["sourceID"] + ".geojson"
	source := props["algoName"]
	version := props["version"]

	dataID, err = pzsvc.Ingest( fName, "geojson", pzAddr, source, version, pzAuth, b2, props)

	return dataID, err
}







