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
	//"net/http"
	//"strings"
	"time"

	//"github.com/venicegeo/pzsvc-lib"
	"gopkg.in/redis.v3"
)

type dBaseClient interface {
	Set(string, interface{}, time.Duration) *redis.StatusCmd
	LPush(string, ...string) *redis.IntCmd
	RPopLPush(string, string) *redis.StringCmd
	GetSet(string, interface{}) *redis.StringCmd
	LRem(string, int64, interface{}) *redis.IntCmd
	Get(key string) *redis.StringCmd
	RPop(key string) *redis.StringCmd
	Del(keys ...string) *redis.IntCmd
}

type mockRedisClient struct {
}

func (mrc *mockRedisClient) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return &redis.StatusCmd{}
}
func (mrc *mockRedisClient) LPush(key string, values ...string) *redis.IntCmd {
	return &redis.IntCmd{}

}
func (mrc *mockRedisClient) RPopLPush(source, destination string) *redis.StringCmd {
	return &redis.StringCmd{}

}
func (mrc *mockRedisClient) GetSet(key string, value interface{}) *redis.StringCmd {
	return &redis.StringCmd{}

}
func (mrc *mockRedisClient) LRem(key string, count int64, value interface{}) *redis.IntCmd {
	return &redis.IntCmd{}

}
func (mrc *mockRedisClient) Get(key string) *redis.StringCmd {
	return &redis.StringCmd{}

}
func (mrc *mockRedisClient) RPop(key string) *redis.StringCmd {
	return &redis.StringCmd{}

}
func (mrc *mockRedisClient) Del(keys ...string) *redis.IntCmd {
	return &redis.IntCmd{}
}
