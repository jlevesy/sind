# go-sind

[![Build Status](https://semaphoreci.com/api/v1/jlevesy/go-sind/branches/master/badge.svg)](https://semaphoreci.com/jlevesy/go-sind)

go-SIND enables you to create swarm clusters on a docker host using SIND (swarm in docker).

Not yet ready to use, this is a PoC at the moment.

## Requirements

- A reachable docker daemon.
- go 1.11.x

## Using it as a go package

Head to the [base example](./example/base/main.go)  or to the [integration test suite](./integration/sind_test.go) to get started.

## Using it as a CLI

### Installation

* Download the latest release for your platform from https://github.com/jlevesy/go-sind/releases
* Untar the downloaded archive and move the binary somewhere on your PATH

```
$ curl -sSL -o sind.tar.gz "https://github.com/jlevesy/go-sind/releases/download/v0.1.0/sind_0.1.0_$(uname -s)_$(uname -m).tar.gz"

$ tar xzf ./sind.tar.gz
$ chmod a+x ./sind
$ mv ./sind /usr/local/bin/
```

### Usage

```
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

## TODO list

- [ ] CI
- [ ] Release process
