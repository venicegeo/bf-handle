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
	//"encoding/json"
	//	"errors"
	"fmt"
	//"io/ioutil"
	//"log"
	"net"
	"net/http"
	//	"os"
	//	"strconv"
	//"strings"
	//"sync"
	"testing"
	"time"

	//"github.com/venicegeo/pzsvc-image-catalog/catalog"
	"github.com/venicegeo/pzsvc-lib"
	"gopkg.in/redis.v3"
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

var mockConnOutpBytes [][]byte

type mockAddr struct{}

func (ma mockAddr) Network() string {
	return "Network"
}

func (ma mockAddr) String() string {
	return "String"
}

type mockConn struct {
	readCount *int
}

var mockConnCount int
var mockConnInst = mockConn{readCount: &mockConnCount}

/*
now we need a series of functions that will populate the
outpBytes (iterating writeCount with each line) based on
the ints, strings, and errors we want to see in each case.

For that, we first need the byte format objective.
Digging commences.

In particular, we're interested in what all touches the
Read port of the net.Conn from teh Dialer in the options.

- all of it (other htan a couple of tests) goes through redis.getDialer in redis.v3/options.go
- That *only* gets referenced in options.go/newConnPool
- This, in turn, only sees the light fo day as the return for `(p *ConnPool) dial()`,
  ...which feeds straight into `(p *ConnPool) NewConn()`
- In the end, it essentially just gets pumped right back out of the redis.Conn Read and Write functions.


*/

func (mCn mockConn) Read(b []byte) (n int, err error) {
	fmt.Printf("reading: %d of %d.\n", *mCn.readCount, len(mockConnOutpBytes))
	if *mCn.readCount < len(mockConnOutpBytes) {
		copy(b, mockConnOutpBytes[*mCn.readCount])
		*mCn.readCount = *mCn.readCount + 1
	}
	return len(b), nil
}

func (mCn mockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (mCn mockConn) Close() error {
	return nil
}

func (mCn mockConn) LocalAddr() net.Addr {
	return mockAddr{}
}

func (mCn mockConn) RemoteAddr() net.Addr {
	return mockAddr{}
}

func (mCn mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mCn mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mCn mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func MockDialer() (net.Conn, error) {
	//build correct net.Conn here.
	return mockConnInst, nil
}

func makeMockRedisCli(outputs []string) *redis.Client {
	opt := redis.Options{Dialer: MockDialer}
	cli := redis.NewClient(&opt)
	mockConnOutpBytes = make([][]byte, len(outputs), len(outputs))
	for i, output := range outputs {
		mockConnOutpBytes[i] = []byte(output)
	}
	return cli
}

func redisConvInt(val int) string {
	return fmt.Sprintf(":%d\r\n", val)
}
func redisConvStatus(val string) string {
	return fmt.Sprintf("+%s\r\n", val)
}
func redisConvString(val string) string {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
}
func redisConvErrStr(val string) string {
	return fmt.Sprintf("-%s\r\n", val)
}

func TestHandleAsynch(t *testing.T) {}

func TestAddAsynchJob(t *testing.T) {
	mockConnCount = 0
	outputs := []string{
		redisConvErrStr("Error: totally an error."),
		redisConvStatus("Pending"),
		redisConvInt(5),
		redisConvStatus("Pending")}
	redisCli = makeMockRedisCli(outputs)
	taskChan = make(chan string)
	w, _, outInt := pzsvc.GetMockResponseWriter()
	r := http.Request{}
	r.Method = "POST"
	r.Body = pzsvc.GetMockReadCloser("string goes here\n")
	addAsynchJob(w, &r)
	if *outInt == http.StatusOK {
		t.Error(`TestAddAsynchJob: passed on what should have been an error return.`)
	}
	addAsynchJob(w, &r)
	if *outInt != http.StatusOK {
		t.Error(`TestAddAsynchJob: failed on what should have been a good run.`)
	}

}

func TestGetAsynchStatus(t *testing.T) {
	mockConnCount = 0
	outputs := []string{
		redisConvErrStr("Error: totally an error."),
		redisConvStatus("Syntax error"),
		redisConvStatus("Pending")}
	redisCli = makeMockRedisCli(outputs)
	w, outStr, outInt := pzsvc.GetMockResponseWriter()
	getAsynchStatus(w, "aaaa")
	if *outInt == http.StatusOK {
		t.Error(`TestGetAsynchStatus: passed on what should have been an error return.  Outmsg: ` + *outStr)
	}
	getAsynchStatus(w, "aaaa")
	if *outInt == http.StatusOK {
		t.Error(`TestGetAsynchStatus: passed on what should have been an error return.  Outmsg: ` + *outStr)
	}
	getAsynchStatus(w, "aaaa")
	if *outInt != http.StatusOK {
		t.Error(`TestGetAsynchStatus: failed on what should have been a good run.  Outmsg: ` + *outStr)
	}
}

func TestGetAsynchResults(t *testing.T) {}
