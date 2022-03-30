# Usage Report

## Record usage

* Usage data will be collected in ClientSideGateway node.
* The data will be collected while user connecting to node and request to service
* The collected data contains those data
  * User ID
  * Service ID
  * Connect count
  * Upload size
  * Download size
* The recorded data is saved as hash table, the key is user key, and the value is the collected data.

## Upload

* A separated loop process/thread is running to summarize report data
* All the data on the node will be collected and uploaded
* (Opt) The data can be encrypted with pre-configured key before upload to improve security
* The summarized report will be uploaded to IPFS node
* Then the IPFS hash and some metadata will be submitted to contract and save on chain
  * The metadata should contain node id, time range 
  * The contract should be able to query the hash list via various conditions
* (Opt) If the raw data should be viewed by user or service provider, there should be separated file pinned on IPFS to avoid information leak

## Generate Report

* Report generator talk with contract to get hash list
  * The request sent to contract should contain query conditions
* Contract returns IPFS hash list
* Generator get files from IPFS and extract needed information
  * Parse the files downloaded
  * Filter out the data in specified time range
  * Load files content and generate user report data and service statistical data