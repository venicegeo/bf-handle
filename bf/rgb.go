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
	"fmt"

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-lib"
)

// rgbGen is designed to work as a subthread function.  It takes in
// a basic input object, provisions appropriate files out of the band
// information, and applies some manner of bandmerge to them (currently
// only pzsvc-ossim is available).  The results get pushed back through
// the given channel.
func rgbGen(inpObj gsInpStruct, rgbChan chan string) {
	var (
		err       error
		fileID    string
		outStruct *pzse.OutStruct
	)

	switch inpObj.BndMrgType {
	case "pzsvc-ossim":

		outFName := "rgb.TIF"
		bands := inpObj.MetaJSON.Properties.Bands
		funcStr := fmt.Sprintf(`bandmerge --output-radiometry U8 --red red.TIF --green green.TIF --blue blue.TIF %s`,
			outFName)

		execObj := pzse.InpStruct{Command: funcStr,
			InExtFiles: []string{0: bands["red"], 1: bands["green"], 2: bands["blue"]},
			InExtNames: []string{0: "red.TIF", 1: "green.TIF", 2: "blue.TIF"},
			OutTiffs:   []string{0: outFName},
			OutTxts:    nil,
			PzAuth:     inpObj.PzAuth}

		outStruct, err = pzse.CallPzsvcExec(&execObj, inpObj.BndMrgURL)
		if err != nil {
			rgbChan <- fmt.Sprintf(`Error: CallPzsvcExec: %s`, err.Error())
			return
		}
		fileID = outStruct.OutFiles[outFName]
		if fileID == "" {
			rgbChan <- fmt.Sprintf(`Error: CallPzsvcExec: No Outfile.  Pzsvc-exec errors: %s`, err.Error())
			return
		}
		fmt.Println("RGB fileId: " + fileID)

	default:
		rgbChan <- ("Error: Unknown bandmerge algorithm")
		return
	}

	outpObj, err := pzsvc.DeployToGeoServer(fileID, "", inpObj.PzAddr, inpObj.PzAuth)

	fmt.Println("RGB geoserver ID: " + outpObj.Layer)

	rgbChan <- outpObj.Layer
	return
}
