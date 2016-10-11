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
	"net/http"

	"github.com/venicegeo/pzsvc-lib"
)

/*
Basic idea: this file is for managing bf-handle job results for the ui.
It's an important part of reusing job runs so that we don't have to
reprocess them all the time.
*/

type sceneInpStruct struct {
	SceneID string `json:"sceneId"`
	PzAddr  string `json:"pzAddr"`
	PzAuth  string `json:"pzAuthToken"`
}

type sceneOutpStruct struct {
	DataIDs []string `json:"dataIds"`
}

// resultsBySceneID takes a sceneID (as per pzsvc-image-catalog) and the necessary information
// for accessing Piazza, and returns a list of bf-handle results in the form of dataIds.
func resultsBySceneID(sceneID, pzAddr, pzAuth string) ([]string, error) {

	files := pzsvc.FileDataList{}
	queryStr := `{"query":{"bool":{"must":[{"match":{"dataResource.dataType.content":"` +
		sceneID +
		`"}},{"match":{"dataResource.dataType.type":"text"}}]}}}`

	_, err := pzsvc.RequestKnownJSON("POST", queryStr, pzAddr+"/data/query", pzAuth, &files)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}

	outDataIds := make([]string, len(files.Data))
	for i, val := range files.Data {
		outDataIds[i] = val.DataID
	}
	return outDataIds, nil
}

// ResultsByScene ...
func ResultsByScene(w http.ResponseWriter, r *http.Request) {
	var inpObj sceneInpStruct
	byts, err := pzsvc.ReadBodyJSON(&inpObj, r.Body)
	if err != nil {
		handleOut(w, "Error: pzsvc.ReadBodyJSON: "+err.Error()+".\nInput String: "+string(byts), nil, http.StatusBadRequest)
		return
	}

	outDataIds, err := resultsBySceneID(inpObj.SceneID, inpObj.PzAddr, inpObj.PzAuth)
	outObj := sceneOutpStruct{DataIDs: outDataIds}
	if err != nil {
		handleOut(w, "resultsByImageID error: "+err.Error(), outObj, http.StatusInternalServerError)
		return
	}

	handleOut(w, "", outObj, http.StatusOK)

}
