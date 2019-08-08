# Contributing to Sake

### Tools

These are tools used for development.

#### Vendoring

`dep` is used to track and vendor dependencies. Before modifying dependencies, install with:

```
$ go get -u github.com/golang/dep/cmd/dep
```

For details on usage, see [the project's github](https://github.com/golang/dep).

#### Dependency Injection

`wire` is used to generate compile-time helpers for wiring up components in a DI-like way. Before building, install with:

```
$ go get -u github.com/google/wire/cmd/wire
```

For details on usage, see [the project's github](https://github.com/google/wire).

## Building

Building the project is straightforward.

#### Dev/Local build

Use `go` and build from the root of the project:

```
$ go build
```

Please note the version number displayed will be the value of `main.DEFAULT_VERSION`

#### Local versioned build

Use `make` to create a versioned build:

```
$ make compile
```

The default version is a semver-compatible string made up of the contents of the `/VERSION` file and the short form of the current git hash (e.g: `1.0.0-c63076f`). To override this default version, you may use the `BUILD_VERSION` environment variable to set it manually:

```
$ BUILD_VERSION=7.7.7-lucky make compile
```

Setting `NO_REV` will not append the git hash as the version label:

```
$ NO_REV=1 make compile
```

When `NO_REV` is used and `REV` is set, you may override the default version label:

```
$ NO_REV=1 REV=lucky make compile
```

#### Dist build

This is primarily meant to be used when building the docker image. Distribution builds are versioned like the local versioned builds and are statically linked(`CGO_ENABLED=0`).

```
$ make dist
```

#### Docker Image

Building a Docker Image is a two-step process. First we build the distribution binary:

```
$ make dist
```

And then we can make the image:

```
$ make image
```

The default image repo used is that of the Makefile's `IMAGE_REPO` variable. The image tag is the `BUILD_VERSION` variable and can be overridden as noted in the *"Local versioned build"* section above. The default `BUILD_VERSION` used

## Testing

Use `make` to run tests:

```
$ make test
```

You can also use `go test` directly for any package without additional bootstrapping:

```
$ go test ./pkg/api/
```
