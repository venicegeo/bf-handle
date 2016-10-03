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
	//	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//	"os"
	//	"strconv"
	"strings"
	"sync"

	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
	"gopkg.in/redis.v3"
)

/*** IMPORTANT NOTES:

It looks like pzsvc-preflight is quashing GET calls.  That's... probably a bad thing, given that it's necessary
for asynch to work.  Confirm with David and/or research, then possibly discuss with Jeff

Need to follow up on whether or not we can make bf-handle guaranteed single-instance.  If so, we're fine.  If not,
that's going to make the current plan about channel and running set and recovering from crashes kind of problematic.


Still to do:
- finish PrepAsynch
- write asynchWorker


*/

var taskChan chan string
var redisCli *redis.Client
var once sync.Once

// HandleAsynch determines which of the asynch functions is appropriate for the given
// call, and does a bit of work extracting information from the requests to simplify
// things downstream and check for obvious errors.  On its first time through, it also
// calls prepAsynch(), and blocks appropriately to make sure that prepAsynch is done
// before anything else happens.  It is the only externally accessible function in
// asynch.go.
func HandleAsynch(w http.ResponseWriter, r *http.Request) {
	once.Do(prepAsynch) // makes sure that all the prep work is done, once, before any other uses of asynch.
	pathStrs := strings.Split(r.URL.Path, "/")
	if len(pathStrs) == 2 {
		addAsynchJob(w, r)
		return
	}
	if len(pathStrs) != 4 {
		pzsvc.HTTPOut(w, `{"Errors": "Incorrect path length for bf-handle asynch.",  "Given Path":"`+r.URL.Path+`"}`, http.StatusBadRequest)
		return
	}
	switch pathStrs[2] {
	case "status":
		getAsynchStatus(w, pathStrs[3])
	case "results":
		getAsynchResults(w, pathStrs[3])
	default:
		pzsvc.HTTPOut(w, `{"Errors": "Not a valid path for bf-handle asynch.",  "Given Path":"`+r.URL.Path+`"}`, http.StatusBadRequest)
	}
}

// addAsynchJob places a new request on the job queue, stores the associated
// input data, makes sure that a worker thread is woken up if any are waiting,
// and sends the jobId to the client.  Input format here shoudl be identical
// to the base bf-handle '/execute' call.
func addAsynchJob(w http.ResponseWriter, r *http.Request) {
	var (
		jobID string
		err   error
		byts  []byte
	)

	jobID, err = pzsvc.PsuUUID()
	if err != nil {
		// failure indicating that the built-in random number generator has run out of bits.
		pzsvc.HTTPOut(w, `{"error":"failure in rand() call", "details":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	byts, err = ioutil.ReadAll(r.Body)
	if err != nil {
		// failure on reading initial call
		errStr := `{"error":"could not understand request", "details":"` + err.Error() + `"}`
		log.Print(pzsvc.TraceStr(errStr))
		pzsvc.HTTPOut(w, errStr, http.StatusInternalServerError)
		return
	}

	err = redisAddJob(jobID, string(byts))
	if err != nil {
		// failure on redis access
		errStr := `{"error":"database access failure", "details":"` + err.Error() + `"}`
		log.Print(pzsvc.TraceStr(errStr))
		pzsvc.HTTPOut(w, errStr, http.StatusInternalServerError)
		return
	}

	select { // this is what a nonblocking unlock looks like in go.
	case taskChan <- "":
	default:
	}
	pzsvc.HTTPOut(w, `{"type":"job","data":{"jobId":"`+jobID+`"}}`, http.StatusOK)
}

// getAsynchStatus grabs the current status of the given job out of redis and
// sends it to the writer
// Acceptable statuses: Pending, Running, Success, Cancelled, Error, Fail
func getAsynchStatus(w http.ResponseWriter, jobID string) {
	statStr, err := redisGetStatus(jobID)
	if err != nil {
		statStr = `{"status":"Error","result" : {"type": "error","message": "Error while retrieving status","details": "Initial error: ` + err.Error() + `"}}`
		pzsvc.HTTPOut(w, `{"Errors":"`+err.Error()+`", "status":"Error" }`, http.StatusInternalServerError)
	}
	if statStr == "Syntax error" {
		statStr = `{"type": "error", "message": "Job not found: ` + jobID + `"}`
	}
	pzsvc.HTTPOut(w, statStr, http.StatusOK)
}

// getAsynchResults grabs the results of a completed job out of redis and
// sends it to the writer
// result format here shoudl be identical to the base bf-handle '/execute' call
func getAsynchResults(w http.ResponseWriter, jobID string) {
	outpStr, err := redisGetResults(jobID)
	if err != nil {
		pzsvc.HTTPOut(w, `{"Errors":"`+err.Error()+`" }`, http.StatusInternalServerError)
	}

	pzsvc.HTTPOut(w, outpStr, http.StatusOK)

}

//
//
//
func asynchWorker(name string) {
	var (
		jobID, inpStr, errStr string
		err                   error
		inpObj                *gsInpStruct
		outpObj               *gsOutpStruct
		outByts               []byte
	)
	fmt.Println("worker " + name + " started")
	for {
		fmt.Println("worker " + name + " begin cycle")
		jobID, inpStr, err = redisTakeJob()
		if jobID == "" {
			fmt.Println("worker " + name + " no job.  Waiting for next job.")
			if err != nil && err.Error() != "redis: nil" {
				errStr = `{"error":"database access failure", "details":"` + err.Error() + `"}`
				log.Print(pzsvc.TraceStr(errStr))
			}
			<-taskChan
			continue
		}
		fmt.Println("worker " + name + " grabs jobID " + jobID)

		inpObj = new(gsInpStruct)
		err = json.Unmarshal([]byte(inpStr), inpObj)
		if err != nil {
			errStr = `{"error":"json unmarshaling error", "details":"` + err.Error() + `"}`
			log.Print(pzsvc.TraceStr(errStr))
			redisErrorJob(jobID, errStr)
			continue
		}
		outpObj, _ = processScene(inpObj)
		if outpObj.Error != "" {
			errStr = pzsvc.TraceStr(`{"error":"scene processing error", "details":"` + outpObj.Error + `"}`)
			log.Print(errStr)
			redisErrorJob(jobID, errStr)
			continue
		}

		outByts, err = json.Marshal(outpObj)
		if err != nil {
			errStr = `{"error":"json marshaling error", "details":"` + err.Error() + `"}`
			log.Print(pzsvc.TraceStr(errStr))
			redisErrorJob(jobID, errStr)
			continue
		}

		redisDoneJob(jobID, string(outByts))
	}
}

// PrepAsynch gets the asynch system up and running.  It checks to see if there are any
// current jobs that were
func prepAsynch() {
	var err error

	redisCli, err = catalog.RedisClient()
	if err != nil {
		log.Println("Failure in Redis call.  Error: " + pzsvc.TraceStr(err.Error()))
	}

	taskChan = make(chan string)
	redisClearDeadJobs()

	go asynchWorker("A")
	go asynchWorker("B")
	go asynchWorker("C")
	// This is somewhat kludgy, and should be fixed later.  For the current system,
	// three worker threads is about right.  While this does create goroutines that
	// are not directly closable, this shouldn't be a leak issue - PrepAsynch is only
	// meant to be called once, and the three routines are intended to last as long
	// as the instance of bf-handle does.
}

const inpLoc = "bf-handle:asynchExecInp:"
const outpLoc = "bf-handle:asynchExecOutp:"
const statusLoc = "bf-handle:asynchExecStatus:"
const jobsLoc = "bf-handle:asynchJobsToDo:"
const runningLoc = "bf-handle:asynchCurrentJobs:"

//
//
//
//
func redisAddJob(jobID, inpObj string) error {
	fmt.Println("Adding Job #" + jobID + ".")
	dataObj := redisCli.Set(inpLoc+jobID, inpObj, 0)
	if dataObj.Err() != nil {
		return dataObj.Err()
	}
	idObj := redisCli.LPush(jobsLoc, jobID)
	if idObj.Err() != nil {
		redisCli.Set(inpLoc+jobID, "", 0)
		return idObj.Err()
	}
	redisCli.Set(statusLoc+jobID, `{"status":"Pending"}`, 0)
	return nil // failure to set status is not logic-breaking
}

// redisTakeJob handles the redis side of a worker thread picking
// up a job from the queue.  It moves the jobID from the "Pending"
// queue to the "Running" queue, grabs and then clears the input
// data, and returns jobID and input data in that order to the
// Callign function.  It will return the empty string and no error
// if there are no jobs in the queue.
func redisTakeJob() (string, string, error) {
	jobObj := redisCli.RPopLPush(jobsLoc, runningLoc)
	jobID := jobObj.Val()
	fmt.Println("Job #" + jobID + " retrieved!")
	if jobID == "" || jobObj.Err() != nil {
		return "", "", jobObj.Err()
	}
	jobDataObj := redisCli.GetSet(inpLoc+jobID, "")
	redisCli.Set(statusLoc+jobID, `{"status":"Running"}`, 0)
	return jobID, jobDataObj.Val(), jobDataObj.Err()
}

//
//
//
// output is set before status to ensure that users who
// receive a status of "Success" are guaranteed to receive
// an output.
func redisDoneJob(jobID, output string) error {
	doneObj := redisCli.LRem(runningLoc, 0, jobID)
	redisCli.Set(outpLoc+jobID, output, 0)
	redisCli.Set(statusLoc+jobID, `{"status":"Success"}`, 0)
	return doneObj.Err()
}

// redisErrorJob is used to try to clean up after a processing error.
// by its nature, it is an attempt to fail out.  As such, the ability
// to respond meaningfully to further failures is limited.
func redisErrorJob(jobID, errString string) {
	errMsg := `{"status":"Error", "result" : {"type": "error","message": "process failure","details": "` + errString + `"}}`
	redisCli.Set(statusLoc+jobID, errMsg, 0)
	redisCli.LRem(runningLoc, 0, jobID)
}

//
//
//
func redisGetStatus(jobID string) (string, error) {
	statusObj := redisCli.Get(statusLoc + jobID)
	return statusObj.Val(), statusObj.Err()
}

//
//
//
func redisGetResults(jobID string) (string, error) {
	resultObj := redisCli.Get(outpLoc + jobID)
	return resultObj.Val(), resultObj.Err()
}

// redisClearDeadJobs
// Does not need to worry about thread-safety. This should only
// ever be called on startup, when there are no other threads to interfere
func redisClearDeadJobs() {
	errMsg := `{"status":"Error", "result" : {"type": "error","message": "crash-interrupt","details": "bf-handle crashed while processing and was rebooted."}}`
	for jobID := redisCli.RPop(runningLoc).Val(); jobID != ""; jobID = redisCli.RPop(runningLoc).Val() {
		redisCli.Set(statusLoc+jobID, errMsg, 0)
		redisCli.Del(inpLoc + jobID)
	}
}
