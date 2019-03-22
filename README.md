# sind

[![CircleCI](https://circleci.com/gh/jlevesy/sind.svg?style=svg)](https://circleci.com/gh/jlevesy/sind)
[![Go Report Card](https://goreportcard.com/badge/github.com/jlevesy/sind)](https://goreportcard.com/report/github.com/jlevesy/sind)

`sind` enables you to create swarm clusters on a docker host using SIND (swarm in docker).

## Requirements

- A reachable docker daemon.
- go 1.11.x

## Using it as a go package

Head to the [example](./cmd/example/main.go)  or to the [integration test suite](./pkg/test) to get started.

## Using it as a CLI

### Installation

- Download the latest release for your platform from <https://github.com/jlevesy/sind/releases>
- Untar the downloaded archive and move the binary somewhere on your PATH

```shell
curl -sSL -o sind.tar.gz "https://github.com/jlevesy/go-sind/releases/download/v0.2.1/sind_0.2.1_$(uname -s)_$(uname -m).tar.gz"

tar xzf ./sind.tar.gz
chmod a+x ./sind
mv ./sind /usr/local/bin/
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
