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
At some point, some of this stuff might get close enough to complete to be worth
pushing over to pzsvc-lib.  For the moment, though, it's kind of crippled - focused
heavily on the specific requirements of dealing with triggers, and specifically on
the way that bf-handle wants to deal with triggers.  The antipathy that Go has
for polymorphism hurts here, given how enthusiastically polymorphic both the
elasticsearch grammar and the piazza backend are in some places
*/


type JobTypeInterface struct {
    Content     string      `json:"content,omitempty"`
    Type        string      `json:"type,omitempty"`
    MimeType    string      `json:"mimeType,omitempty"`
}

type JobData struct{
    ServiceID       string                      `json:"serviceId,omitempty"`
    DataInputs      map[string]JobTypeInterface `json:"dataInputs,omitempty"`
    DataOutput      []JobTypeInterface          `json:"dataOutput,omitempty"`
}

type TrigJob struct{
    JobType     struct{
        Type    string      `json:"type,omitempty"`
        Data    JobData     `json:"data,omitempty"`
    }                       `json:"jobType,omitempty"`
}

type CompClause struct {
    LTE     interface{}     `json:"lte,omitempty"`
    GTE     interface{}     `json:"gte,omitempty"`
    Format  string          `json:"format,omitempty"`
}

type QueryClause struct {
    Match   map[string]string       `json:"match,omitempty"`
    Range   map[string]CompClause   `json:"range,omitempty"`
}

type TrigQuery struct{
    Bool    struct{
        Filter  []QueryClause   `json:"filter"`
    }                           `json:"bool"`
}

type TrigCondition struct{
    EventTypeIDs        []string        `json:"eventTypeIds"`
    Query   struct{
                Query   TrigQuery       `json:"query"`
            }                           `json:"query"`
}

type TrigStruct struct {
    Name            string	        `json:"name"`
    Enabled         bool            `json:"enabled"`
    Condition       TrigCondition   `json:"condition"`
    Job             TrigJob         `json:"job"`
}
