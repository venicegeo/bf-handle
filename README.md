#bf-handle

"bf-handle" is designed to provide a simple interface to the bf-ui for interacting with a variety of potential image analysis services, a variety of potential image sources, and the Piazza framework itself.  Currently the only service interface it handles is pzsvc-ossim, but we intend to add to that list as additional services are deveoped.

## Installing and Running

bf-handle is relatively straightforward.  It can be installed via go install.  When run from the command line without further parameters, it will begin to serve from the local host.  If the PORT environment variable is specified, it will use that.  Otherwise it will default to 8085.  If you wish to provide an auth token for piazza, it should be at the environment variable BFH_PZ_AUTH.  If you wish to provide an auth token for external database access, it should be at the environment variable BFH_DB_AUTH.

bf-handle does not currently have an autoregistration feature.  To register the service to Piazza, please see appropriate piazza documentation.

## Service Call Format By Endpoint

Usage notes:
All bf-handle inputs and outputs are json objects.  The following should be interpreted accordingly.  Any case where pzAuthToken is referenced, the string required is the exact string that goes into the "Authorization" header for calls to the local piazza gateway.  In any case where pzAddr is required, it should begin at "https://" and it should not have a trailing slash.

### bf-handle/execute

The primary purpose of bf-handle execute is managing image analysis services on behalf of the beachfront UI.  It accepts an input json object, reaches out to the specified services and data sources, and produces a result in the form of a geojson file uploaded to the local Piazza instance and a json response.  The format of the input is as follows:
```
algoType      string    // API for the shoreline algorithm
svcURL        string    // URL for the shoreline algorithm
tideURL       string    // optional.  URL for the tide service (optional)
metaDataJSON	Feature		// semi-optional.  Entry from Image Catalog
metaDataURL   string		// semi-optional.  URL for the Image Catalog
bands         []string  // names of bands to feed into the shoreline algorithm
pzAuthToken   string    // semi-optional.  Auth string for this Pz instance
pzAddr        string    // gateway URL for this Pz instance
dbAuthToken   string    // semi-optional.  Auth string for the image database
lGroupId      string    // UUID string for the target geoserver layer group
jobName       string    // Arbitrary user-defined name string for resulting job
```

A more detailed explanation for each follows:

"algoType" is the type of the algorithm that you intend to call.  From this, we derive the necessary inputs and expected outputs for that algorithm.  Currently only supports "pzsvc-ossim".

"svcURL" is the URL of the algorithm service you intend to call.  If you are using Piazza, this should be easy to acquire from the service listing.

"tideURL" is the URL of the tide information service.  If it is provided, bf-handle will call it and add the results to the metadata for each feature of the resulting geojson.  Currently, only github/venicegeo/bf_TidePrediction is supported as a format

"metaDataJSON" and "metaDataURL" are both in reference to pzsvc-image-catalog.  One or the other is required, but both would be redundant.  pzsvc-image-catalog provides geojson features in a specific format in response to an image search, each representing a particular scene.  bf-handle execute requires one such feature per run.  "metaDataJSON" expects the feature itself, while "metaDataURL" expects a URL that will return the feature in question.  pzsvc-image-catalog does serve those, if an instance is available.

"bands": a comma-separated list (no spaces) of band names for the frequency ranges you want to include.  Reference only as many bands as you wish fed into the algorithm.  Band names can be drawn from the list of available bands listed in the "metaDataJSON" field.  For the moment, the preferred bands to feed into pzsvc-ossime are "coastal" and "swir1".

"pzAuthToken": overrides the authorization token for Piazza access.  If not provided, will default to the contents of BFH_PZ_AUTH (if any).

"pzAddr": the address of the local piazza instance.  Necessary for things like ingesting image files and updating response metadata.

"dbAuthToken": overrides the authorization token for external database access.  If not provided, will default to the contents of BFH_DB_AUTH (if any).

"1GroupId": references Geoserver.  When provided, provisions the result geojson to geoserver and adds the resulting geoserver layer to the layer group with the given ID.  If the given ID does not currently exist as a layer group, will create a layer group with that ID and with the resulting geoserver layer as its first element.

"jobName": an arbitrary string.  Will be added on to job response as the property "jobName".  Primarily meant as a tool for simplifying result searches and/or UI labeling.

//------

bf-handle responds with a json string including the following:
- "shoreDataID": a dataID for the S3 bucket of the piazza instance provided in "pzAddr".  That dataID will contain the geoJSON result of the algorithm call, plus a significant amount of metadata.

- "shoreDeplID": a deployment ID for a layer in the geoserver instance associated with the targeted piazza instance, also containing the output data.

- "geometry": provides the boundaries of the detection in geojson format.

- "algoType": value copied from the input parameter of the same name.

- "sceneCaptureDate": the date/time that the images for the scene were taken.

- "sceneId": the ID in pzsvc-image-catalog that references the scene used.

- "jobName": value copied from the input parameter of the same name.

- "sensorName": the name of the source for the original scene.  Indicates things like which bands were available, the zoom level, and so forth.

- "svcURL": value copied from the input parameter of the same name.

- "error": describes any errors that may have occurred during processing.

### bf-handle/executeBatch

This endpoint is designed to support the detection of a large geographic area. It does the following:
1. Produces a set of footprints based on the optimum scenes available (considering cloud cover, date of acquisition, and optionally the tide service)
1. Ingests the footprints and publishes them as a WFS
1. Executes shoreline detection on each scene
1. Assembles the shorelines into a single product, correlating to the baseline and clipping the results to the footprints
1. Ingests the shorelines and publishes them as a WFS

You can execute it through a curl script like the following:

```
curl -X POST -S -s \
  -H 'Content-Type: application/json' \
    -o response.txt \
    -d @$1 \
    "http://localhost:8085/executeBatch"

```

The input file is a JSON object containing the following properties:

* baseline: a GeoJSON object (usually a FeatureCollection) containing the baseline
* footprintsID: the Piazza ID of the footprints for the optimum scenes (if you have this, you don't need a baseline)
* algoType: something like "pzsvc-ossim"
* svcURL The OSSIM URL, e.g., "https://pzsvc-ossim.stage.geointservices.io/execute"
* pzAuthToken: something like "Basic JKLDJKLDFkl3jKLFDLK2JKkHDKJHHAI=="
* pzAddr: something like: "https://pz-gateway.stage.geointservices.io"
* dbAuthToken: a hex token provided by Piazza
* bands: ["coastal","swir1"]
* tidesAddr: location of the tide prediction service (optional), e.g., "https://TidePrediction.stage.geointservices.io/tides"

This process will issue events to report its progress:
* :beachfront:executeBatch:footprintsIngested
* :beachfront:executeBatch:completed
* :beachfront:executeBatch:failed

See /eventTypes below to find the Event Type IDs for these Event Types. This process will ingest both the footprints and the detected shorelines into Piazza. The footprints ID will come back as a return to the service call. Since the actual detection 


### bf-handle/prepareFootprints

...

### bf-handle/assembleShorelines

...

### bf-handle/newProductLine

bf-handle/newProductLine creates a Beachfront Product Line.  A product line consists of a Pz trigger, calling bf-handle/execute, using a given eventTypeId and event filter, and associated with a new geoserver layer group.  Once this trigger is created, it will run bf-handle/execute every time an event fires on that event type that passes the filter, and then push the result into geoserver in the given layer group.

Input Format:
```
bfInputJSON	  *	      // this is an object.  It's format is that of the input data for the '/execute' call.
maxx		      float	  // Part of bounding box.  Required.
minx		      float	  // Part of bounding box.  Required.
maxy		      float	  // Part of bounding box.  Required.
miny		      float	  // Part of bounding box.  Required.
cloudCover	  float	  // Max allowed cloud cover.  '10' would permit cloud cover of up to 10%.  Required.
maxRes		    string	// Max allowed resolution.  '30' would represent 30 meter resolution.
minRes		    string	// Min allowed resolution.  As above.
maxDate		    string	// No images taken after this date will be processed.  "yyyy-MM-dd'T'HH:mm:ssZZ" format
minDate		    string	// No images taken before this date will be processed.  Required.
sensorName  	string	// Name of the sensor producing the data - 'landsat', for example.  Used for search and display.
eventTypeId	  string	// Piazza Event Type ID for pzsvc-image-catalog's "new image" Event Type
serviceId	    string	// Piazza Service ID for bf-handle
name		      string	// Arbitrary name for the product line.  Intended for display
```
Output Format:
```
triggerId	    string	// Piazza Trigger ID for the newly created trigger
layerGroupId	string	// Layer Group ID for the associated geoserver layer group
```
Currently, the geoserver layer group does not exist until the first image comes in through the product line.  Once it does exist, it will contain all images from the product line.

### bf-handle/getProductLines

bf-handle/getProductLines allows returns a list of product lines, filtered by creator ID.

Input format:
```
eventTypeId   string	// Piazza event type ID from pzsvc-image-catalog/eventTypeID.  Indicates a newly cataloged scene.
serviceId	    string	// Piazza service ID for bf-handle's /execute endpoint.
createdBy	    string	// Username of the person that created this product line.  Filter.
pzAddr		    string	// Gateway URL for this Pz instance
pzAuthToken	  string	// Auth string for this Pz instance
sortBy		    string	// which output parameter to sort by 
order	      	string	// whether that parameter should be sorted ascending (asc) or descending (desc) 
```
Output format:
```
productLines	*	      // this is a list of JSON objects.  Those objects are in the input format for the '/newProductLines' endpoint 
```

### bf-handle/eventTypes

This is a simple endpoint that returns all known Event Types as a JSON object. When an event type is needed, you provide a root and the system does some version checking, adding a new Event Type if needed.

### bf-handle/resultsByScene

The API for this endpoint is temporary.  It is likely to be modified within the next month to improve information output.  Currently, it takes a pzsvc-image-catalog sceneId, and returns a list of all jobs that have been run against that scene in the form of Pz DataIds.

Input format:
```
sceneID		    string	// the ID in pzsvc-image-catalog that references the scene used
pzAddr		    string	// the gateway URL for this Pz instance
pzAuthToken	  string	// the auth string for this Pz instance
```

Output format:
```
dataIds		string list	// Pz dataIds.  These are the dataIds resulting from successful job calls.  The files they point to are the bf-handle /execute output strings.
```

### bf-handle/resultsByProductLine

This API is temporary.  In the long term, we expect to modify it heavily, improving both searchability and information output.  It is possible that this endpoint will be closed, and replaced with one or more new endpoints.  Currently, it allows you to specify a given product line (trigger), and returns the jobs that have been triggered by that Product Line, in the form of a paginated list of dataIds.

Input format:
```
TriggerID	    string	// the ID for the trigger/Product Line.
PerPage		    string	// the number of jobIds to list per "page"
PageNo		    string	// the number of pages of that size to skip before beginning to list 
PzAddr		    string	// the gateway URL for this Pz instance
PzAuthToken	  string	// the auth string for this Pz instance
```

Output format:
unlike the rest of the entries on this page, resultsByProductLine just returns a json-marshaled list of strings, rather than and object.  Those strings are the same sorts of dataIds returned by the resultsByScene call.
