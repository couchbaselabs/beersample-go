## Couchbase Beer Tutorial

This tutorial uses Go in combination with Couchbase Server to
display and manage beers and breweries found in the beer-sample
dataset.


### Quick Start

Install Couchbase Server: http://www.couchbase.com/download

During initial configuration load `beer-sample` dataset. It could be
also loaded later if your node/cluster has free space. More info
http://www.couchbase.com/docs/couchbase-manual-2.0/couchbase-sampledata-beer.html

Add two more views to your bucket. To save prevous design documents
you should first copy your design document from "Production Views" tab
to "Development Views", update it and publish back to production. Here
are two new views you need:

* Design document: `_design/beer`. View: `by_name`

```javascript
function (doc, meta) {
  if(doc.type && doc.type == "beer") {
    emit(doc.name, null);
  }
}
```
* Design document: `_design/brewery`. View: `by_name`

```javascript
function (doc, meta) {
  if(doc.type && doc.type == "brewery") {
    emit(doc.name, null);
  }
}
```

Clone this repo:

    $ git clone git://github.com/couchbaselabs/beersample-go.git
    Cloning into 'beersample-go'...

Run the application.

    $ go run
    Starting server on :9980

Navigate to http://localhost:9980/welcome.