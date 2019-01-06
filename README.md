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
- [ ] Build a cool CLI on top of that like `sind new --worker=4 --managers=3 -p 4999:4999`
