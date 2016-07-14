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
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"log"
//	"net/http"
//	"os"
	
//	"github.com/venicegeo/pzsvc-lib"
//	"github.com/venicegeo/geojson-go/geojson"
//	"github.com/venicegeo/pzsvc-image-catalog/catalog"
)



/*
	what the user has going in...
	- They know what database they want things from.
	- They know what filters they want to apply.
	- They know the bf-handle call they want to make on it (other than the imagecatalog data)
	- They know the EventTypeID

	by the time it gets here...
	- The exact proccessing command(s) will have been established.

	things we can/should derive for ourselves...
	- the layer ID for geoserver ingest (added later)
	- the triggerId (output.  importance?)
	- the actual call to bf-handle that gets triggered
	  - somehow get the new image data from imagecatalog
	    - in the event?
		- ID in the event, get for ourselves?  (would have to also get contact info for image handler)

*/