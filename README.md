# Sake: a saga coordinator

Sake is an orchestration service that handles the execution, interpretation, and recovery of distributed sagas in a microservice environment.

## Installation

```sh
go get github.com/danielkrainas/sake
```

## Usage

```sh
sake run -c <config_path>
```

## Project Status

Sake is currently in alpha stage development and **not** intended for production use at this time.

## Bugs and Feedback

If you see a bug or have a suggestion, feel free to open an issue [here](https://github.com/danielkrainas/sake/issues).

## Contributions

PR's welcome! There are no strict style guidelines, just follow best practices and try to keep with the general look & feel of the code present. All submissions must pass `golint` and have a test to verify *(if applicable)*.

## License

[Unlicense](http://unlicense.org/UNLICENSE). This is a Public Domain work.

[![Public Domain](https://licensebuttons.net/p/mark/1.0/88x31.png)](http://questioncopyright.org/promise)

> ["Make art not law"](http://questioncopyright.org/make_art_not_law_interview) -Nina Paley
