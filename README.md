# go-sind

go-SIND enables you to create swarm clusters on a docker host using SIND (swarm in docker).

Not yet ready to use, this is a PoC at the moment.

## Requirements

- A reachable docker daemon.
- go 1.11.x

## Getting started

Head to the [base example](./example/base/main.go)  or to the [integration test suite](./integration/sind_test.go) to get started.

## Why ?

Mostly for automated testing.

## TODO list

- [ ] CI.
- [ ] Randomize port binding to ensure that multiple clusters can be started on the same host concurrently.
- [ ] Make sure that it works on a remote docker host.
