# Visualizer

Visualizer, what's based on the library [Pixel](https://github.com/faiface/pixel), is a main-thread that visualizes stuff.

## Prerequisite

*You might want to use MSYS or MSYS2 to run bash commands on Windows systems.*

#### Have Go installed properly

```bash
$ export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
$ echo "PATH=$PATH:$GOROOT/bin:$GOPATH/bin" >> ~/.bashrc
$ echo "PATH=$PATH:$GOROOT/bin:$GOPATH/bin" >> ~/.bash_profile
```

#### Install dependencies of amidakuji package

```bash
$ go get -v github.com/nanitefactory/amidakuji/...
$ go get -v -u github.com/go-bindata/go-bindata/...
```

```bash
$ cd $GOPATH/src/github.com/nanitefactory/amidakuji
$ make glossary/asset.go
```

## Installation

```bash
$ go get -v github.com/nanitefactory/visual
```

## Getting started

```go
import "github.com/nanitefactory/visual"
```

(I'll add some examples and write ups later xD)
