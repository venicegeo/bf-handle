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

type imageInpStruct struct {
	ImageID	string	`json:"imageId"`
	PzAddr	string	`json:"pzAddr"`
	PzAuth	string	`json:"pzAuth"`
}

// resultsByImageId takes an imageId (as per pzsvc-image-catalog) and the necessary information
// for accessing Piazza, and returns a list of bf-handle results in the form of dataIds.
func resultsByImageID(imageID, pzAddr, pzAuth string) ([]string, error) {
	
	files := pzsvc.FileDataList{}
	
	queryStr := `{"query":{"bool":{"must":[{"match":{"dataResource.dataType.content":"` +
				imageID +
				`"}},{"match":{"dataResource.dataType.type":"text"}}]}}}`
	
	_, err := pzsvc.RequestKnownJSON("POST", queryStr, pzAddr, pzAuth, files)
	if err != nil {
		return nil, pzsvc.AddRef(err)
	}
	outDataIds := make([]string, len(files.Data))
	for i := range (files.Data) {
		outDataIds[i] = files.Data[i].DataID
	}
	return outDataIds, nil
}


// ResultsByImage ...
func ResultsByImage (w http.ResponseWriter, r *http.Request)  {
	
}

// 4 *** struct for daterange max/min(X/Y) and other search parameters
//    - the better this lines up with search requirements for the Image Service, the better.
//    - correction: there's no good reason to define this here.  We should jsut use whatever
//      they use for image catalog.
//    - currently, they use a series of form fields, rather than an input Json.  Given that and
//      a few other details of implementation, would probably be worthwhile to refactor as part
//      of implementing this
//      - be sure to discuss with Jeff first.  Would not do to be rude.
//    - alternate version: some way of adding a "has been processed" filter to the image catalog
//      - would want to be handled on the fly - records might be lost or imagecatalog might be
//        moved or some such
//      - On the flip side, pagination with an erratic filter is a *hassle*.  Basically the only
//        way around that without depaginating and then repaginating (ugh) would be to have the
//        filter be part of the searchable data (which, honestly, would also save time)

// 3 *** function for taking a triggerID and returning a list of job outptus goes here (probably)

// 6 *** function for taking a set of search params (see #5) and returning a list of job outputs goes here.
