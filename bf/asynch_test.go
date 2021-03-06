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
	"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
	"net/http"
	"testing"
)

/*

so, right now, we can only get blank response objects.  Either we do what we can with those, or
we build another interface workaround.

so... what if we ran a multi-stage build thing?
- we can't use actual redis.Client objects.  We can't define them properly, and they're not interfaces.
We'd need a dbaseClient interface that redis.Client qualified for.
- this needs to have functions.
- those functions need to return actual redis library objects.
- we can't create said redis library objects.

By extension, then, we'd need to create our own objects that served as a wrapper to the redis objects,
or re-implement the redis objects ourselves.

On the bright side, we can specify a nonstandard client for redis, using a list of options - and that
list of options lets us specify a net.Conn, which we can abuse to hand it whatver feedback we like.
The only real problem remaining at that point is to chase everythign through to find out what sort
of information coming out of the connection would lead to correct output fromt eh redis client.  It's
not insoluble, but it is nontrivial

To somewhat simplify things, we should put together a straightforward function that generates the
client inputs necessary to get the outputs we want.  We can write the subfunction, test it reasonably
trivially (as part of the go test run), tweak it as necessary, and then not have to deal with it again.
Once we have that ugliness black-boxed, figuring out what we want to have come out of it is pretty easy.

*/

func TestHandleAsynch(t *testing.T) {
}

func TestAddAsynchJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	outputs := []string{
		catalog.RedisConvErrStr("Error: totally an error."),
		catalog.RedisConvStatus("Pending"),
		catalog.RedisConvInt(5),
		catalog.RedisConvStatus("Pending")}
	redisCli = catalog.MakeMockRedisCli(outputs)
	taskChan = make(chan string)
	w, outstr, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser("string goes here\n")

	addAsynchJob(w, &r)
	t.Log(*outstr)
	if *outInt == http.StatusOK {
		t.Error(`TestAddAsynchJob: passed on what should have been an error return.`)
	}
	addAsynchJob(w, &r)
	t.Log(*outstr)
	if *outInt != http.StatusOK {
		t.Error(`TestAddAsynchJob: failed on what should have been a good run.`)
	}

}

func TestGetAsynchStatus(t *testing.T) {
	catalog.SetMockConnCount(0)
	outputs := []string{
		catalog.RedisConvErrStr("Error: totally an error."),
		catalog.RedisConvStatus("Syntax error"),
		catalog.RedisConvStatus("Pending")}
	redisCli = catalog.MakeMockRedisCli(outputs)
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	getAsynchStatus(w, "aaaa")
	if *outInt == http.StatusOK {
		t.Log(`TestGetAsynchStatus: passed on what should have been an error return.  Outmsg: ` + *outStr)
	}
	getAsynchStatus(w, "aaaa")
	if *outInt == http.StatusOK {
		t.Log(`TestGetAsynchStatus: passed on what should have been an error return.  Outmsg: ` + *outStr)
	}
	getAsynchStatus(w, "aaaa")
	if *outInt != http.StatusOK {
		t.Log(`TestGetAsynchStatus: failed on what should have been a good run.  Outmsg: ` + *outStr)
	}
}

func TestGetAsynchResults(t *testing.T) {
	catalog.SetMockConnCount(0)
	outputs := []string{
		catalog.RedisConvErrStr("Error: totally an error."),
		catalog.RedisConvStatus("Pending"),
		catalog.RedisConvStatus("Finished")}
	redisCli = catalog.MakeMockRedisCli(outputs)
	w, _, _ := pzsvc.GetMockResponseWriter()
	getAsynchResults(w, "aaaa")
	getAsynchResults(w, "aaaa")
	getAsynchResults(w, "aaaa")
}
func TestRedisAddJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	outputs := []string{
		catalog.RedisConvErrStr("123")}
	redisCli = catalog.MakeMockRedisCli(outputs)
	redisAddJob("123", "Test")
}
func TestRedisTakeJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	outputs := []string{
		catalog.RedisConvString("123"),
		catalog.RedisConvStatus("Pending")}
	redisCli = catalog.MakeMockRedisCli(outputs)
	out1, out2, _ := redisTakeJob()
	t.Log(out1)
	t.Log(out2)
}
func TestRedisDoneJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	_ = redisDoneJob("123", "test")
}

func TestRedisErrorJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	redisErrorJob("123", "test")
}

func TestRedisClearJob(t *testing.T) {
	catalog.SetMockConnCount(0)
	_ = redisClearJob("123")
}

func TestRedisCloseDeadJobs(t *testing.T) {
	redisCloseDeadJobs()
}
func TestPrepAsynch(t *testing.T) {
	prepAsynch()
}
