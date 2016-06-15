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

func proc (w http.ResponseWriter, r *http.Request) {
	var inpObj struct {
		AlgoType	string			`json:"algoType"`
		AlgoURL		string			`json:"svcURL"`
		MetaJSON	geojson.Feature	`json:"metaDataJSON"`
		Bands		[]string		`json:"bands"`
		PzAuth		string			`json:"pzAuthToken"`
		PzAddr		string			`json:"pzAddr"`
		DbAuth		string			`json:"dbAuthToken"`
	}
	fmt.Println ("bf-handle called.")		
	inpBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintln(w, "Error: ioutil.ReadAll: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}	
	
	err = json.Unmarshal(inpBytes, &inpObj)
	if err != nil {
		fmt.Fprintln(w, "Error: json.Unmarshal: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}

	(&inpObj.MetaJSON).ResolveGeometry()
	
	if inpObj.PzAuth == "" {
		inpObj.PzAuth = os.Getenv("BFH_PZ_AUTH")
	}
	
	if inpObj.DbAuth == "" {
		inpObj.DbAuth = os.Getenv("BFH_DB_AUTH")
	}

	fmt.Println ("bf-handle: provisioning begins.")
	dataIDs, err := provision(&inpObj.MetaJSON, inpObj.DbAuth, inpObj.PzAuth, inpObj.PzAddr, inpObj.Bands)
	if err != nil{
		fmt.Fprintln(w, "Error: bf-handle provisioning: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}

	fmt.Println ("bf-handle: running Algo")
	resDataID, err := runAlgo(inpObj.AlgoType, inpObj.AlgoURL, inpObj.PzAuth, dataIDs)
	if err != nil{
		fmt.Fprintln(w, "Error: algo result: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
	}
	
	fmt.Println (`bf-handle: getting metadata ( dataId = ` + resDataID + `)`)	
	attMap, err := getS3Meta (resDataID, inpObj.PzAddr, inpObj.PzAuth, &inpObj.MetaJSON)
	if err != nil{
		fmt.Fprintln(w, "Error: bf-handle get metadata: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

// **** use the following to update S3 metadata for ossim-like algos
//	err = pzsvc.UpdateFileMeta(resDataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
//	if err != nil {
//		fmt.Fprintln(w, "Error: bf-handle update s3 metadata: " + err.Error())
//		w.WriteHeader(http.StatusInternalServerError)
//	}

	resDataID, err = addGeoFeatureMeta(resDataID, inpObj.PzAddr, inpObj.PzAuth, attMap)
	if err != nil {
		fmt.Fprintln(w, "Error: bf-handle update feature metadata: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	fmt.Println ("outputting")
	fmt.Fprintf(w, resDataID)
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.
func provision(metaDataFeature *geojson.Feature, dbAuth, pzAuth, pzAddr string, bands []string) ( []string, error ) {
	
	dataIDs := make([]string, len(bands))

	fSource := metaDataFeature.PropertyString("sensorName")

	for i, band := range bands {
		reader, err := catalog.ImageFeatureIOReader(metaDataFeature, band, dbAuth)
		if err != nil {
			return nil, err
		}
		fName := fmt.Sprintf("%s-%s.TIF", fSource, band)

		bSlice, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		// TODO: at some point, we might wish to add properties to the TIFF files as we ingest them.
		// We'd do that by replacing the "nil", below, with an appropriate map[string]string.
		dataID, err := pzsvc.Ingest(fName, "raster", pzAddr, fSource, "", pzAuth, bSlice, nil)
		if err != nil {
			return nil, err
		}
		dataIDs[i] = dataID
	}
	return dataIDs, nil
}

func runAlgo( algoType, algoURL, authKey string, dataIDs []string) (string, error) {
	switch algoType {
	case "pzsvc-ossim":
		return runOssim (algoURL, dataIDs[0], dataIDs[1], authKey)
	default:
		return "", fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, algoType)
	}
}

// runOssim does all of the things necessary to process the given images
// through pzsvc-ossim.  It constructs and executes the request, reads
// the response, and extracts the dataID of the output from it.
func runOssim(algoURL, imgID1, imgID2, authKey string) (string, error) {
	type execStruct struct {
		InFiles		map[string]string
		OutFiles	map[string]string
		ProgReturn	string
		Errors		[]string
	}
	
	imgName1 := (imgID1 + ".TIF")
	imgName2 := (imgID2 + ".TIF")
	geoJName := "shoreline.geojson"
	funcStr := fmt.Sprintf(`shoreline --image %s,%s --projection geo-scaled --threshold 0.5 --tolerance 0 %s`,
							imgName1, imgName2, geoJName)
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

// new format:
// ossim-cli shoreline -i garden_b3.tif, garden_b6.tif  --prop prop1:myprop1 --prop prop2:myprop2 debug.json
// prop1:myprop1 and prop2:myprop2 are arbitrary key/value pairs for this purpose.  They'll be added to the properties.
// the filenames (garden_b3.tif and garden_b6.tif) will also be saved in an "input_files" list.  Figure out a way to
// make that work well.  Talk with Mark about what sort of information would fit well into that space.

}

// updateS3Meta grabs the existing metadata for a file from the S3 bucket, adds the metadata that
// can be gleaned from the geojson feature provided by the imag harvester, and returns the result.
func getS3Meta(dataID, pzAddr, pzAuth string, feature *geojson.Feature) (map[string]string, error) {
	dataRes, err := pzsvc.GetFileMeta(dataID, pzAddr, pzAuth)
	if err != nil {
		return nil, err
	}

	attMap := make(map[string]string)
	for key, val := range dataRes.Metadata.Metadata {
		attMap[key] = val
	}
	
	attMap["sourceID"] = feature.ID // covers source and ID in that source
	attMap["dateTimeCollect"] = feature.PropertyString("acquiredDate")
	attMap["sensorName"] = feature.PropertyString("sensorName")
	attMap["resolution"] = feature.PropertyString("resolution")
	
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
// TODO: fill in some of those empty quotes.
	dataID, err = pzsvc.Ingest("", "geojson", pzAddr, "", "", pzAuth, b2, props)

	return dataID, err
}







