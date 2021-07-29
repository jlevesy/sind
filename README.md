# sind

[![Go Report Card](https://goreportcard.com/badge/github.com/jlevesy/sind)](https://goreportcard.com/report/github.com/jlevesy/sind)

`sind` enables you to create swarm clusters on a docker host using SIND (swarm in docker).

## Using it as a go package

Head to the [example](./cmd/example/main.go)  or to the [integration test suite](./pkg/test) to get started.

## Using it as a CLI

### Installation

```shell
curl -sSL https://raw.githubusercontent.com/jlevesy/sind/master/install.sh | bash
```

### Usage

```shell
# Will create a new cluster with 3 managers, 3 workers  and the port 8080 of the host bound
# to the port 8080 of the ingress network of the cluster.
sind create --managers=3 --workers=3 -p 8080:8080

# Setup the docker cli configuration to communicate with the new cluster.
eval $(sind env)

# Deploy an app
docker stack deploy -c my-stack.yml app

# Enjoy your app :)
docker service ls

# Once your're done, clear your docker CLI configuration then delete your cluster
unset DOCKER_HOST
sind delete
```

## Why ?

Mostly for automated testing.
