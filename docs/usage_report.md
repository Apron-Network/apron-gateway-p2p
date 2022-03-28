# Usage Report

## Record usage

### User

* Service usage data of each user will be collected at ClientSideGateway node.
* The collected data contains those data
  * User ID
  * Service ID
  * Connect count
  * Upload size
  * Download size

### Service provider

* Service usage data of each service will be collected at ServiceSideGateway node. 
* The data should have similar format with collected user data

## Upload

* A separated loop process/thread is running to summarize report data
* All the data on the node, includes user data (ClientSideGateway) and service data (ServiceSideGateway) will be collected
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