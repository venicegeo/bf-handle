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
//	"encoding/json"
	"fmt"
//	"io/ioutil"
//	"log"
//	"net/http"
//	"os"
	
	"github.com/venicegeo/pzsvc-lib"
//	"github.com/venicegeo/geojson-go/geojson"
//	"github.com/venicegeo/pzsvc-image-catalog/catalog"
)


func rgbGen(inpObj inpStruct, rgbChan chan string) {
	bandIDs, err := provision(inpObj, []string{"red","green","blue"})
	if err != nil {
		rgbChan <- ("Error: " + err.Error())
		return
	}
	var fileID string

	switch inpObj.BndMrgType {
	case "pzsvc-ossim":

		outFName := "rgb.TIF"

		funcStr := fmt.Sprintf(`bandmerge --red %s --green %s --blue %s %s`,
								bandIDs[0] + ".TIF",
								bandIDs[1] + ".TIF",
								bandIDs[2] + ".TIF",
								outFName)

		execObj := pzsvc.ExecIn{FuncStr:funcStr,
								InFiles:bandIDs,
								OutGeoJSON:nil,
								OutGeoTIFF:[]string{0:outFName},
								OutTxt:nil,
								AlgoURL:inpObj.BndMrgURL,
								AuthKey:inpObj.PzAuth}

		outMap, err := pzsvc.CallPzsvcExec(&execObj)
		if err != nil {
			rgbChan <- fmt.Sprintf(`Error: CallPzsvcExec: %s`, err.Error())
			return
		}
		fileID = outMap[outFName]
		fmt.Println("RGB fileId: " + fileID)

	default:
		rgbChan <- ("Error: Unknown bandmerge algorithm")
		return
	}

	outpID, err := pzsvc.DeployToGeoServer(fileID, inpObj.PzAddr, inpObj.PzAuth)

	fmt.Println("RGB geoserver ID: " + outpID)

	rgbChan <- outpID
	return
}