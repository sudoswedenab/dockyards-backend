# dockyards-backend

This README would normally document whatever steps are necessary to get your application up and running.

## What is this repository for?

Creating, reading and managing rancher clusters.

# Build

## Building locally

To build locally the golang compiler needs to be installed. Follow the upstream installation documentation found [here](https://go.dev/doc/install).

```
$ go build ./cmd/backend
```

## Building with Docker

Project is using standard golang container image from docker hub, check [here](https://hub.docker.com/_/golang).

```
$ docker build -t dockyards-backend .
$ docker run -it --rm dockyards-backend
```

A compose file is also available that will start a local database and rancher environment.

```
$ docker-compose up --detach
```


# Configuration

## Configuration flags

|name|description|default|
|---|---|---|
|`-del-garbage-interval`|delete garbage interval, in seconds|`60`|
|`-log-level`|log level (debug, info, warn or error)|`info`|
|`-trust-insecure`|trust insecure certificates|`false`|
| `-use-inmem-db`|use in-memory database (all data is temporary)|`false`|

## Configuration environment variables

Urls and secrets to the following services are needed:

- Postgres database
- Rancher
- OpenStack

All settings related to these services are configured using environment variables. As a convenience any environment variables can be placed in a [.env](https://github.com/joho/godotenv#usage) file in the root of the project.

### Postgres environment variables

|name|description|example|
|---|---|---|
|`DB_CONF`|database configuration string|`host=localhost user=postgres password=notverysecure dbname=dockyards`|

or separately

|name|description|example|
|---|---|---|
|`DB_HOST`|database host|`localhost`|
|`DB_PORT`|database port|`5432`|
|`DB_USER`|database user|`postgres`|
|`DB_PASSWORD`|database password|`notverysecure`|
|`DB_NAME`| database name|`dockyards`|

### Rancher environment variables

|name|description|example|
|---|---|---|
|`CATTLE_URL`|full rancher url|`https://my-rancher.cluster:8443/v3`|
|`CATTLE_BEARER_TOKEN`|rancher authorization token|`token-abc123:dG9rZW4tYWJjMTIz`|

### OpenStack environment variables

|name|description|example|
|---|---|---|
|`OPENSTACK_AUTH_URL`|full openstack identity url|`https://my-openstack.cluster:5000/v3`|
|`OPENSTACK_APP_ID`|application credential id|`9d967ec81f7`|
|`OPENSTACK_APP_SECRET`|application credential secret|`aHR0cHM6Ly9teS1`|

### Copied Code

Some code was copied from https://github.com/dgrijalva/jwt-go/blob/master/hmac_example_test.go which is under MIT license.

### Contribution guidelines

* Writing tests
* Code review
* Other guidelines

### Who do I talk to?

* Repo owner or admin
* Other community or team contact
