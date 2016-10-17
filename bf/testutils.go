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
import (
	//"encoding/json"
	//"net/http"
	//"strings"
	"time"
	//"github.com/venicegeo/pzsvc-lib"
	//"gopkg.in/redis.v3"
)

type dBaseClient interface {
	Set(string, interface{}, time.Duration) interface {
		Err() error
		Val() string
	}
	LPush(string, ...string) interface {
		Err() error
		Val() int64
	}
	RPopLPush(string, string) interface {
		Err() error
		Val() string
	}
	GetSet(string, interface{}) interface {
		Err() error
		Val() string
	}
	LRem(string, int64, interface{}) interface {
		Err() error
		Val() int64
	}
	Get(key string) interface {
		Err() error
		Val() string
	}
	RPop(key string) interface {
		Err() error
		Val() string
	}
	Del(keys ...string) interface {
		Err() error
		Val() int64
	}
}

type strValOut interface {
	Err() error
	Val() string
}

type intValOut interface {
	Err() error
	Val() int64
}

type bytValOut interface {
	Err() error
	Val() []byte
}

type mockStrVal struct {
	err error
	val string
}

func (mock mockStrVal) Err() error  { return mock.err }
func (mock mockStrVal) Val() string { return mock.val }

type mockIntVal struct {
	err error
	val int64
}

func (mock mockIntVal) Err() error { return mock.err }
func (mock mockIntVal) Val() int64 { return mock.val }

type mockBytVal struct {
	err error
	val []byte
}

func (mock mockBytVal) Err() error  { return mock.err }
func (mock mockBytVal) Val() []byte { return mock.val }

type mockRedisClient struct {
}

func (mrc *mockRedisClient) Set(key string, value interface{}, expiration time.Duration) *mockStrVal {
	return &mockStrVal{}
}
func (mrc *mockRedisClient) LPush(key string, values ...string) *mockIntVal {
	return &mockIntVal{}

}
func (mrc *mockRedisClient) RPopLPush(source, destination string) *mockBytVal {
	return &mockBytVal{}

}
func (mrc *mockRedisClient) GetSet(key string, value interface{}) *mockBytVal {
	return &mockBytVal{}

}
func (mrc *mockRedisClient) LRem(key string, count int64, value interface{}) *mockIntVal {
	return &mockIntVal{}

}
func (mrc *mockRedisClient) Get(key string) *mockBytVal {
	return &mockBytVal{}

}
func (mrc *mockRedisClient) RPop(key string) *mockBytVal {
	return &mockBytVal{}

}
func (mrc *mockRedisClient) Del(keys ...string) *mockIntVal {
	return &mockIntVal{}
}
*/
/**

It looks like it's not actually possible to plug a homebrewed interface of any type
into the same slot as an externally built interface that has functions that return
an externally built interface

We could build an interface wrapper around the redis interfaces but that's some serious
additional code ugliness there.  We could reduce that ugliness a bit by removing the
effective polymorphism - have it be a wrapper that runs a mock if given nothing, or
Redis if given a redis client.  It's still a block of extra code ugliness, but it's not
*as* bad.

So... various plans so far that might mostly work:
- have a mock redis client that hands out empty redis objects.  Test as far as that
will let you test and no further.
- Build a mighty wrapper arnough redis.  That wrapper contains the redis client pointer
as an internal variable, and defines all fo the functions we care about.  When redisClient
is non-null, it hands back results fo calls ot redisClient.  When redisClient is null,
it hands back mock data.
- ignore asynch entirely for right now.  Make up the numbers elsewhere.

**/
