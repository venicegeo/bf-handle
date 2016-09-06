#bf-handle

"bf-handle" is designed to provide a simple interface to the bf-ui for interacting with a variety of potential image analysis services, a variety of potential image sources, and the Piazza framework itself.  Currently the only service interface it handles is pzsvc-ossim, but we intend to add to that list as additional services are deveoped.

## Installing and Running

bf-handle is relatively straightforward.  It can be installed via go install.  When run from the command line without further parameters, it will begin to serve from the local host.  If the PORT environment variable is specified, it will use that.  Otherwise it will default to 8085.  If you wish to provide an auth token for piazza, it should be at the environment variable BFH_PZ_AUTH.  If you wish to provide an auth token for external database access, it should be at the environment variable BFH_DB_AUTH.

bf-handle does not currently have an autoregistration feature.  To register the service to Piazza, please see appropriate piazza documentation.

## Service Call Format By Endpoint

### bf-handle/execute

The primary purpose of bf-handle execute is managing image analysis services on behalf of the beachfront UI.  It accepts an input json object, reaches out to the specified services and data sources, and produces a result in the form of a geojson file uploaded to the local Piazza instance and a json response.  The format of the input is as follows:

algoType	string		// API for the shoreline algorithm
svcURL		string		// URL for the shoreline algorithm
tideURL		string		// optional.  URL for the tide service (optional)
metaDataJSON	Feature		// semi-optional.  Entry from Image Catalog
metaDataURL	string		// semi-optional.  URL URL for the Image Catalog
bands		string array	// names of bands to feed into the shoreline algorithm
pzAuthToken	string         // semi-optional.  Auth string for this Pz instance
pzAddr		string		// gateway URL for this Pz instance
dbAuthToken	string		// semi-optional.  Auth string for the image database
lGroupId	string		// UUID string for the target geoserver layer group
jobName		string		// Arbitrary user-defined name string for resulting job


A more detailed explanation for each follows:

"algoType" is the type of the algorithm that you intend to call.  From this, we derive the necessary inputs and expected outputs for that algorithm.  Currently only supports "pzsvc-ossim".

"svcURL" is the URL of the algorithm service you intend to call.  If you are using Piazza, this should be easy to acquire from the service listing.

"tideURL" is the URL of the tide information service.  If it is provided, bf-handle will call it and add the results to the metadata for each feature of the resulting geojson.  Currently, only github/venicegeo/bf_TidePrediction is supported as a format

"metaDataJSON" and "metaDataURL" are both in reference to pzsvc-image-catalog.  One or the other is required, but both would be redundant.  pzsvc-image-catalog provides geojson features in a specific format in response to an image search, each representing a particular scene.  bf-handle execute requires one such feature per run.  "metaDataJSON" expects the feature itself, while "metaDataURL" expects a URL that will return the feature in question.  pzsvc-image-catalog does serve those, if an instance is available.

"bands": a comma-separated list (no spaces) of band names for the frequency ranges you want to include.  Reference only as many bands as you wish fed into the algorithm.  Band names can be drawn from the list of available bands listed in the "metaDataJSON" field.  For the moment, the preferred bands to feed into pzsvc-ossime are "coastal" and "swir1".

"pzAuthToken": Overrides the authorization token for Piazza access.  If not provided, will default to the contents of BFH_PZ_AUTH (if any)

"pzAddr": the address of the local piazza instance.  Necessary for things like ingesting image files and updating response metadata 

"dbAuthToken": Overrides the authorization token for external database access.  If not provided, will default to the contents of BFH_DB_AUTH (if any)

"1GroupId": References Geoserver.  When provided, provisions the result geojson to geoserver and adds the resulting geoserver layer to the layer group with the given ID.  If the given ID does not currently exist as a layer group, will create a layer group with that ID and with the resulting geoserver layer as its first element

"jobName": An arbitrary string.  Will be added on to job response as the property "jobName".  Primarily meant as a tool for simplifying result searches and/or UI labeling.

//------

bf-handle responds with a json string including the following:
- "shoreDataID": a dataID for the S3 bucket of the piazza instance provided in "pzAddr".  That dataID will contain the geoJSON result of the algorithm call, plus a significant amount of metadata.

- "shoreDeplID": a deployment ID for a layer in the geoserver instance associated with the targeted piazza instance, also containing the output data.

- "rgbLoc": Piazza S3 bucket dataID for the results of the bandmerge algorithm (if it was requested)

- "geometry": Provides the boundaries of the detection in geojson format

- "algoType": Value copied from the input parameter of the same name.

- "sceneCaptureDate": The date/time that the images for the scene were taken.

- "sceneId": The ID in pzsvc-image-catalog that references the scene used.

- "jobName": Value copied from the input parameter of the same name.

- "sensorName": The name of the source of the original scene.  Indicates things like which bands were available, the zoom level, and so forth.

- "svcURL": Value copied from the input parameter of the same name.

- "error": describes any errors that may have occurred during processing

### bf-handle/newProductLine

bf-handle/newProductLine creates a trigger into bf-handle/execute, using the given eventTypeId and event filter, and associates it with a new geoserver layer group.  Once this trigger is created, it will run bf-handle/execute every time an event fires on that event type that passes the filter, and then push the result into geoserver in the given layer group.
