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
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	
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
	
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func proc (w http.ResponseWriter, r *http.Request) {
	algoType := r.FormValue("algoType")
	algoURL := r.FormValue("svcURL")
	metaJSON := r.FormValue("metaDataJSON")
	bands := strings.Split(r.FormValue("bands"), ",")
	pzAuth := r.FormValue("pzAuthToken")
	pzAddr := r.FormValue("pzAddr")
	dbAuth := r.FormValue("dbAuthToken")
	
	if pzAuth == "" {
		pzAuth = os.Getenv("PZ_AUTH")
	}
	
	if dbAuth == "" {
		dbAuth = os.Getenv("DB_AUTH")
	}
	
	dataIDs, err := provision(metaJSON, dbAuth, pzAuth, pzAddr, bands)
	if err != nil{
		fmt.Println("Error: bf-handle provisioning: " + err.Error())
	}
	fmt.Println ("running Algo")	
	resDataID, err := runAlgo(algoType, algoURL, dataIDs)
	if err != nil{
		fmt.Println("Error: algo result: " + err.Error())
	}
	
	fmt.Println (`updating Data ( dataId = ` + resDataID + `)`)	
	err = updateData (resDataID, pzAddr, pzAuth, metaJSON)
	if err != nil{
		fmt.Println("Error: bf-handle update data: " + err.Error())
	}	
	fmt.Println ("outputting")	
	fmt.Fprintf(w, resDataID)
}

// provision uses the given image metadata to access the database where its image set is stored,
// download the images from that image set associated with the given bands, upload them to
// the S3 bucket in Pz, and return the dataIds as a string slice, maintaining the order from the
// band string slice.
func provision(metaDataJSON, dbAuth, pzAuth, pzAddr string, bands []string) ( []string, error ) {
	
	dataIDs := make([]string, len(bands))
	
	metaDataFeature, err := geojson.FeatureFromBytes( []byte(metaDataJSON) )
	if err != nil {
		return nil, err
	}

	fSource := metaDataFeature.PropertyString("sensorName")

	for i, band := range bands {
		
		reader, err := catalog.ImageFeatureIOReader(metaDataFeature, band, dbAuth)
		if err != nil {
			return nil, err
		}

		fName := fmt.Sprintf("%s-%s.TIF", fSource, band)	
		dataID, err := pzsvc.IngestTiffReader(fName, pzAddr, fSource, "", pzAuth, reader, nil)
		if err != nil {
			return nil, err
		}
		dataIDs[i] = dataID
	}
	return dataIDs, nil
}

func runAlgo( algoType, algoURL string, dataIDs []string) (string, error) {
	switch algoType {
	case "pzsvc-ossim":
		return runOssim (algoURL, dataIDs[0], dataIDs[1])
	default:
		return "", fmt.Errorf(`bf-handle error: algorithm type "%s" not defined`, algoType)
	}
}

// runOssim does all of the things necessary to process the given images
// through pzsvc-ossim.  It constructs and executes the request, reads
// the response, and extracts the dataID of the output from it.
func runOssim(algoURL, imgID1, imgID2 string) (string, error) {
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

// updateData modifies the S3 metadata of the given file.  Specifically, it
// adds information on the image source - what external source it was drawn
// from, the image ID at that source, the date/time of image collection, and
// the name of the sensor that did the collecting.
func updateData(dataID, pzAddr, pzAuth, featJSON string) error {
	dataRes, err := pzsvc.GetFileMeta(dataID, pzAddr, pzAuth)
	if err != nil {
		return err
	}

	attMap := make(map[string]string)
	for key, val := range dataRes.Metadata.Metadata {
		attMap[key] = val
	}
	
	feature, err := geojson.FeatureFromBytes([]byte(featJSON))
	if err != nil {
		return err
	}
	
	attMap["sourceID"] = feature.ID // covers source and ID in that source
	attMap["dateTimeCollect"] = feature.PropertyString("acquiredDate")
	attMap["sensorName"] = feature.PropertyString("sensorName")
	err = pzsvc.UpdateFileMeta(dataID, pzAddr, pzAuth, attMap)
	if err != nil {
		return err
	}
	
	return nil
/*
TODO: still want to pass over SRC_RESOLUTION (float)
currently don't knwo where to get it.  Talk with Jeff?
*/

}
