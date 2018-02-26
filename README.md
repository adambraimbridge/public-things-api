[![Circle CI](https://circleci.com/gh/Financial-Times/public-things-api/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/public-things-api/tree/master)[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/public-things-api)](https://goreportcard.com/report/github.com/Financial-Times/public-things-api) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/public-things-api/badge.svg)](https://coveralls.io/github/Financial-Times/public-things-api)

# Public API for Things (public-things-api)
__Provides a public API for Things stored in a Neo4J graph database__

## Installation

Download the source code, dependencies and test dependencies:

```
go get -u github.com/kardianos/govendor
mkdir $GOPATH/src/github.com/Financial-Times/public-things-api
cd $GOPATH/src/github.com/Financial-Times
git clone https://github.com/Financial-Times/public-things-api.git
cd public-things-api && govendor sync
go build .
```

## Running locally

1. Run the tests and install the binary:

    ```
    govendor sync
    govendor test -v -race +local
    go install
    ```

2. Run the binary (using the `help` flag to see the available optional arguments):

    ```
    $GOPATH/bin/public-things-api [--help]

    Options:
        --neo-url="http://localhost:7474/db/data"   neo4j endpoint URL ($NEO_URL)
        --port="8080"                               Port to listen on ($PORT)
        --graphiteTCPAddress=""                     Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally) ($GRAPHITE_ADDRESS)
        --graphitePrefix=""                         Prefix to use. Should start with content, include the environment, and the host name. e.g. coco.pre-prod.public-things-api.1 ($GRAPHITE_PREFIX)
        --logMetrics=false                          Whether to log metrics. Set to true if running locally and you want metrics output ($LOG_METRICS)
        --env="local"                               environment this app is running in
        --cache-duration="30s"                      Duration Get requests should be cached for. e.g. 2h45m would set the max-age value to '7440' seconds ($CACHE_DURATION)
        --logLevel="info"                           Log level of the app ($LOG_LEVEL)
    ```

## Build and deployment

* The application is built as a docker image inside a helm chart to be deployed in a Kubernetes cluster.
  An internal Jenkins job takes care to push the Docker image to Docker Hub and deploy the chart when a tag is created.
  This is the Docker Hub repository: [coco/public-things-api](https://hub.docker.com/r/coco/public-things-api)
* CI provided by CircleCI: [public-things-api](https://circleci.com/gh/Financial-Times/public-things-api)

## Service endpoints

### Getting a "thing" description

Using curl:

```
curl http://localhost:8080/things/{concept-uuid} | jq
```

This an example of the response body

```
{
  "id": "http://api.ft.com/things/a11fa00f-777d-484a-9ebc-fbf81b774fc0",
  "apiUrl": "http://api.ft.com/things/a11fa00f-777d-484a-9ebc-fbf81b774fc0",
  "prefLabel": "Solar Wars",
  "types": [
    "http://www.ft.com/ontology/core/Thing",
    "http://www.ft.com/ontology/concept/Concept",
    "http://www.ft.com/ontology/Topic"
  ],
  "directType": "http://www.ft.com/ontology/Topic",
  "aliases": [
    "Solar Wars"
  ]
}
```

### Getting a "thing" description with its relationships with other concepts

The client can request additional information about specific relationships with other concepts
by adding the `showRelationship` query parameters in the request URL.
The `showRelationship` parameters can assume the following values:
* `broader`, which append all the concepts that have 
a [SKOS broader](https://www.w3.org/2009/08/skos-reference/skos.html#broader) 
relationship with the requested "thing"; 
* `broaderTransitive`, which append all the concepts that have 
a [SKOS broader transitive](https://www.w3.org/2009/08/skos-reference/skos.html#broaderTransitive) 
relationship with the requested "thing"; 
* `narrower`, which append all the concepts that have 
a [SKOS narrower](https://www.w3.org/2009/08/skos-reference/skos.html#narrower) 
relationship with the requested "thing"; 
* `related`, which append all the concepts that have 
a [SKOS related](https://www.w3.org/2009/08/skos-reference/skos.html#related) 
relationship with the requested "thing".

This is an example of curl request:
```
curl http://localhost:8080/things/a11fa00f-777d-484a-9ebc-fbf81b774fc0?showRelationship=broaderTransitive&showRelationship=related&showRelationship=narrower | jq
```

This is a potential response of a thing description with relationships
```
{
  "id": "http://api.ft.com/things/a11fa00f-777d-484a-9ebc-fbf81b774fc0",
  "apiUrl": "http://api.ft.com/things/a11fa00f-777d-484a-9ebc-fbf81b774fc0",
  "prefLabel": "Solar Wars",
  "types": [
    "http://www.ft.com/ontology/core/Thing",
    "http://www.ft.com/ontology/concept/Concept",
    "http://www.ft.com/ontology/Topic"
  ],
  "directType": "http://www.ft.com/ontology/Topic",
  "aliases": [
    "Solar Wars"
  ],
  "narrowerConcepts": [
    {
      "id": "http://api.ft.com/things/0ff1c1c9-970a-4f05-9f97-c5150f8f907e",
      "apiUrl": "http://api.ft.com/things/0ff1c1c9-970a-4f05-9f97-c5150f8f907e",
      "prefLabel": "Macroeconomics",
      "types": [
        "http://www.ft.com/ontology/core/Thing",
        "http://www.ft.com/ontology/concept/Concept",
        "http://www.ft.com/ontology/Topic"
      ],
      "directType": "http://www.ft.com/ontology/Topic",
      "predicate": "http://www.w3.org/2004/02/skos/core#narrower"
    }
  ],  
  "broaderConcepts": [
    {
      "id": "http://api.ft.com/things/49181791-a1a9-4966-ac30-010846ec76d8",
      "apiUrl": "http://api.ft.com/things/49181791-a1a9-4966-ac30-010846ec76d8",
      "prefLabel": "Trade disputes",
      "types": [
        "http://www.ft.com/ontology/core/Thing",
        "http://www.ft.com/ontology/concept/Concept",
        "http://www.ft.com/ontology/Topic"
      ],
      "directType": "http://www.ft.com/ontology/Topic",
      "predicate": "http://www.w3.org/2004/02/skos/core#broader"
    },
    {
      "id": "http://api.ft.com/things/243243d9-de4b-4869-909b-fab711125624",
      "apiUrl": "http://api.ft.com/things/243243d9-de4b-4869-909b-fab711125624",
      "prefLabel": "Global Trade",
      "types": [
        "http://www.ft.com/ontology/core/Thing",
        "http://www.ft.com/ontology/concept/Concept",
        "http://www.ft.com/ontology/Topic"
      ],
      "directType": "http://www.ft.com/ontology/Topic",
      "predicate": "http://www.w3.org/2004/02/skos/core#broaderTransitive"
    }
  ],
  "relatedConcepts": [
    {
      "id": "http://api.ft.com/things/29e9fad1-14fc-480b-a89c-cd964750bd80",
      "apiUrl": "http://api.ft.com/things/29e9fad1-14fc-480b-a89c-cd964750bd80",
      "prefLabel": "Renewable Energy",
      "types": [
        "http://www.ft.com/ontology/core/Thing",
        "http://www.ft.com/ontology/concept/Concept",
        "http://www.ft.com/ontology/Topic"
      ],
      "directType": "http://www.ft.com/ontology/Topic",
      "predicate": "http://www.w3.org/2004/02/skos/core#related"
    },
    {
      "id": "http://api.ft.com/things/86fb0401-ec02-419d-a14e-74078cb8b662",
      "apiUrl": "http://api.ft.com/things/86fb0401-ec02-419d-a14e-74078cb8b662",
      "prefLabel": "Protectionism",
      "types": [
        "http://www.ft.com/ontology/core/Thing",
        "http://www.ft.com/ontology/concept/Concept",
        "http://www.ft.com/ontology/Topic"
      ],
      "directType": "http://www.ft.com/ontology/Topic",
      "predicate": "http://www.w3.org/2004/02/skos/core#related"
    }
  ]
}
```

## Healthchecks

Admin endpoints are:

* `/__gtg`
* `/__health`
* `/__build-info`
* `/__ping`

At the moment the `/__health` and `/__gtg` check the the connectivity to a neo4j database.

### Logging

* The application uses [logrus](https://github.com/sirupsen/logrus); the log file is initialised in [main.go](main.go).
* Logging requires an `env` app parameter, for all environments other than `local` logs are written to file.
* When running locally, logs are written to console. If you want to log locally to file, you need to pass in an env parameter that is != `local`.
* NOTE: `/__build-info` and `/__ping` endpoints are not logged as this information is not needed in logs/splunk.