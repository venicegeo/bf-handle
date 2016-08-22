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

/*
Basic idea: this file is for managing "jobs" as bf-ui considers them.
Specifically, these are the records left behind when bf-handle process
runs.  Currently, we're going to be getting job-by-databaseImageID and
job-by-triggerID
*/

// 1 *** struct for job output goes here
//    - should include createdBy (processedBy?)

// 5 *** struct for daterange max/min(X/Y) and other search parameters goes here
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

// 2 *** function for taking a dbaseID and returning a list of job outputs goes here
// note: the function is very pzsvc-lib in structure, but there are details of implementation that
// are very bf-handle.  Is there a good/worthwhile place to split the two?  (it's not all that big to begin with)
// extension on note: is there a use elsewhere for searches against existing file metadata?
// - takes dbaseID, pzAuth, pzAddr string as input param.  returns output param and error.
// - calls pzAddr + data?keyword= + dbaseID, passing in pzAuth as appropriate (page? pageSize? ordering?)
//   - receives and demarshals pzsvc.FileDataList object.
// - creates empty slice of output objects
// - for range through DataDesc list, filter out any false positives, calls (#2a) on true
//   positives, and append the results to the output slice
// - return output slice

// 2a: function for taking a pzsvc.DataDesc object and returnign a job output (#1) object

// 3 *** function for taking a jobID (an alert object?) and returning a job output goes here

// 4 *** function for taking a triggerID and returning a list of job outptus goes here (probably)

// 6 *** function for taking a set of search params (see #5) and returning a list of job outputs goes here.
