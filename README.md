#bf-handle

"bf-handle" is designed to provide a simple interface to the bf-ui for interacting with a variety of potential image analysis services, a variety of potential image sources, and the Piazza framework itself.  Currently the only service interface it handles is pzsvc-ossim, but we intend to add to that list as additional services are deveoped.

## Installing and Running

bf-handle is relatively straightforward.  It can be installed via go install.  When run from the command line without further parameters, it will begin to serve from the local host.  If the PORT environment variable is specified, it will use that.  Otherwise it will default to 8085.  If you wish to provide an auth token for piazza, it should be at the environment variable BFH_PZ_AUTH.  If you wish to provide an auth token for external database access, it should be at the environment variable BFH_DB_AUTH.

bf-handle does not currently have an autoregistration feature.  To register the service to Piazza, please see appropriate piazza documentation.

## Service Call Format By Endpoint

### bf-handle/execute

bf-handle execute accepts POST calls set up in x-www-form-urlencoded format, with the following form values:

"algoType": the type of the algorithm that you intend to call.  From this, we derive the necessary inputs and expected outputs.  Currently only supports "pzsvc-ossim".

"svcURL": the URL of the algorithm service you intend to call.  If you are using Piaza, should be easy to acquire from the service listing.

"metaDataJSON": a block of metadata describing a single set of images.  Format not described here as it should be exactly the same format as that used by the pz-image-catalog image search.  Select one entry from out of the search results and send in the entire thing.  It should be legal JSON.

"bands": a comma-separated list (no spaces) of band names for the frequency ranges you want to include.  Reference only as many bands as you wish fed into the algorithm.  Band names can be drawn from the list of available bands listed in the "metaDataJSON" field

"pzAuthToken": Overrides the authorization token for Piazza access.  If not provided, will default to the contents of BFH_PZ_AUTH (if any)

"pzAddr": the address of the local piazza instance.  Necessary for things like ingesting image files and updating response metadata 

"dbAuthToken": Overrides the authorization token for external database access.  If not provided, will default to the contents of BFH_DB_AUTH (if any)

"bandMergeType": Supports optional bandmerge/rbg option.  The type/API of bandmerge service you intend to call.  If blank, will skip bandmerge.

"bandMergeURL": Supports optional bandmerge/rbg option.  The URL of the bandmerge service.

bf-handle responds with a json string including the following:
- "shoreDataID": a dataID for the S3 bucket of the piazza instance provided in "pzAddr".  That dataID will contain the geoJSON result of the algorithm call, with a few additional pieces of metadata, noting when the source images were collected, what sensor platform collected them, what database the images were sourced from, and what the image ID was in that database.

- "shoreDeplID": a deployment ID for a layer in the geoserver instance associated with the targeted piazza instance, also containing the output data.

- "rgbLoc": Piazza S3 bucket dataID for the results of the bandmerge algorithm (if requested)

- "error": describes any errors that may have occurred during processing

### bf-handle/newProductLine

creates a trigger into bf-handle/execute and associates it with a geoserver layer group.  Once this product line is created, it will run bf-handle/execute every time an appropriate event fires
