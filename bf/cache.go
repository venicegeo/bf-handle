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
	"log"
	"net/http"
	"time"

	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
	"gopkg.in/redis.v3"
)

var cacheMapSem pzsvc.Semaphore
var cacheMap map[string]*cacheHolder

const waitTime = 360000

type cacheHolder struct {
	sem      pzsvc.Semaphore
	httpStat int
	outp     *gsOutpStruct
}

func cacheInit() {
	cacheMapSem = make(pzsvc.Semaphore, 1)
}

func cachedProcessScene(key string, shouldReadCache bool, inpObj *gsInpStruct) (*gsOutpStruct, int) {

	cacheMapSem.Lock()
	if cacheMap[key] == nil {
		cacheMap[key] = &cacheHolder{}
		cacheMap[key].sem.Lock()
		cacheMapSem.Unlock()
		cacheMap[key].outp, cacheMap[key].httpStat = processScene(inpObj)
		cacheMap[key].sem.Unlock()
		return cacheMap[key].outp, cacheMap[key].httpStat
	}
	if !shouldReadCache {
		return processScene(inpObj)
	}
	cacheMap[key].sem.Lock()
	if cacheMap[key].outp == nil {
		cacheMap[key].outp, cacheMap[key].httpStat = processScene(inpObj)
	}
	cacheMap[key].sem.Unlock()
	return cacheMap[key].outp, cacheMap[key].httpStat
}

// cachedProcessSceneRedis is a way to use redis to coordinate the behavior
// of, potentially, multiple threads run on multiple computers, with the intent
// that only one of them try to process a give job at a given time, and that
// if any of them succeeds, all accept that success and return with it.  If you
// do not understand how multithreaded programmign works, much of this will
// be confusing to you.
func cachedProcessSceneRedis(key string, shouldReadCache bool, inpObj *gsInpStruct) (*gsOutpStruct, int) {
	var (
		inWait    bool
		err       error
		status    int
		outpObj   *gsOutpStruct
		outpJSON  []byte
		timeStr   string
		timeKey   = key + "***" + inpObj.AlgoURL + "***" + inpObj.TideURL + "***" + "time"
		outputKey = key + "***" + inpObj.AlgoURL + "***" + inpObj.TideURL + "***" + "output"
		timeFmt   = "01/02/2006 15:04:05"
	)

	inWaitTime := func(timeLockObj *redis.StringCmd) (bool, error) {
		if timeLockObj.Err() != nil {
			return false, timeLockObj.Err()
		}
		if timeLockObj.Val() == "" {
			return false, nil
		}
		dbTime, err := time.Parse(timeFmt, timeLockObj.Val())
		if err != nil {
			return false, err
		}
		return dbTime.Sub(time.Now()) < 15*time.Minute, nil
	}

	if redisCli == nil {
		if redisCli, err = catalog.RedisClient(); err != nil {
			log.Println("Failure in Redis call.  Error: " + pzsvc.TraceStr(err.Error()))
		}
	}

	// this is the "wait and see" loop.  Exits to this loop should be either by
	// break (when it is discovered that the current thread has priority and should
	// be the processing thread), Process and return (if the connection with redis
	// breaks down) or return (when it is discovered that some other thread has
	// completed and stored a correct response).
	for {
		timeLockObj := redisCli.Get(timeKey)
		inWait, err = inWaitTime(timeLockObj)
		if err != nil {
			log.Println("cachedProcessSceneRedis: Failure in inWaitTime call #1.  Error: " + err.Error())
			return processScene(inpObj)
		}
		if !inWait {
			// this plays around with race conditions a bit, but GetSet is atomic,
			// which means that only one of the listening threads can actually get
			// the blank-and/or-old record (assuming reasonably synchronized clocks)
			// Having the various Now() calls overwrite each other in quick
			// succession shouldn't be an issue as the requirement for 15 minutes
			// is more fo a rough guide than anythign precise.
			timeLockObj = redisCli.GetSet(timeKey, time.Now().Format(timeFmt))
			inWait, err = inWaitTime(timeLockObj)
			if err != nil {
				log.Println("cachedProcessSceneRedis: Failure in inWaitTime call #2.  Error: " + err.Error())
				return processScene(inpObj)
			}
			if !inWait {
				break
			}
		}
		// at this point, we've concluded that this thread isn't the processing thread,
		// or at least not yet.  We'll keep an eye on the output spot until

		for inWait {
			outpRed := redisCli.Get(outputKey)
			if outpRed.Err() != nil {
				log.Println("cachedProcessSceneRedis: Failure in redisCli Get call.  Error: " + err.Error())
				return processScene(inpObj)
			}
			// get from output.  If output exists, respond with status 200
			if outpRed.Val() != "" {
				var outpObj gsOutpStruct
				err = json.Unmarshal([]byte(outpRed.Val()), outpObj)
				if err != nil {
					log.Println("cachedProcessSceneRedis: Failure in output unmarshal.  Error: " +
						err.Error() +
						".  Original bytes: " +
						outpRed.Val())
					return processScene(inpObj)
				}
				return &outpObj, http.StatusOK
			}
			time.Sleep(time.Minute)
			inWait, err = inWaitTime(timeLockObj)
			if err != nil {
				log.Println("cachedProcessSceneRedis: Failure in inWaitTime call #3.  Error: " + err.Error())
				return processScene(inpObj)
			}
		}
	}
	timeStr = time.Now().Format(timeFmt)
	redisCli.Set(timeKey, timeStr, 2*time.Hour)
	outpObj, status = processScene(inpObj)
	outpJSON, err = json.Marshal(outpObj)
	redisCli.Set(outputKey, outpJSON, 2*time.Hour)

	return outpObj, status
}

/*
Path 1: On successful initial lock:
A - write current time to timelock
B - run process
C - write correct answer to output location
D - return correct answer

Path 2: On unsuccessful lock
A - check output location.  If populated, return it
B - check timelock.  If within wait time, wait time (30 seconds?  A minute?), then go to A of Path 2.
C - Attempt Get/Set trylock on timelock.  If resulting Get time is larger than wait time
    (ie, you're the first to grab after it timed out) move to "B" of Path 1.  Else, go to A of Path 2.
*/
/*
need a time format that translates into actual time in some fashion, that we can then use to compare times coherently
*/
