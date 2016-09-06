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
resultName	string		// Arbitrary user-defined name string for result


A more detailed explanation for each follows:

"algoType" is the type of the algorithm that you intend to call.  From this, we derive the necessary inputs and expected outputs for that algorithm.  Currently only supports "pzsvc-ossim".

"svcURL" is the URL of the algorithm service you intend to call.  If you are using Piazza, this should be easy to acquire from the service listing.

"tideURL" is the URL of the tide information service.  If it is provided, bf-handle will call it and add the results to the metadata for each feature of the resulting geojson.  Currently, only github/venicegeo/bf_TidePrediction is supported as a format

"metaDataJSON" and "metaDataURL" are both in reference to pzsvc-image-catalog.  One or the other is required, but both would be redundant.  pzsvc-image-catalog provides geojson features in a specific format in response to an image search, each representing a particular scene.  bf-handle execute requires one such feature per run.  "metaDataJSON" expects the feature itself, while "metaDataURL" expects a URL that will return the feature in question.  pzsvc-image-catalog does serve those, if an instance is available.

"bands": a comma-separated list (no spaces) of band names for the frequency ranges you want to include.  Reference only as many bands as you wish fed into the algorithm.  Band names can be drawn from the list of available bands listed in the "metaDataJSON" field.  For the moment, the preferred bands to feed into pzsvc-ossime are "coastal" and "swir1".

"pzAuthToken": Overrides the authorization token for Piazza access.  If not provided, will default to the contents of BFH_PZ_AUTH (if any)

"pzAddr": the address of the local piazza instance.  Necessary for things like ingesting image files and updating response metadata 

"dbAuthToken": Overrides the authorization token for external database access.  If not provided, will default to the contents of BFH_DB_AUTH (if any)

"1GroupId":

"resultName":




bf-handle responds with a json string including the following:
- "shoreDataID": a dataID for the S3 bucket of the piazza instance provided in "pzAddr".  That dataID will contain the geoJSON result of the algorithm call, with a few additional pieces of metadata, noting when the source images were collected, what sensor platform collected them, what database the images were sourced from, and what the image ID was in that database.

- "shoreDeplID": a deployment ID for a layer in the geoserver instance associated with the targeted piazza instance, also containing the output data.

- "rgbLoc": Piazza S3 bucket dataID for the results of the bandmerge algorithm (if requested)

- "error": describes any errors that may have occurred during processing

### bf-handle/newProductLine

creates a trigger into bf-handle/execute and associates it with a geoserver layer group.  Once this product line is created, it will run bf-handle/execute every time an appropriate event fires
