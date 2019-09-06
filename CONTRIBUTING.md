## Contributing

Contributions by anyone are welcome.
You can help by expanding the code, implementing more test cases, or just using the compiler and reporting bugs.

### Coding Style

We do not enforce a strict coding style in this project.
However, we try to write code that is documented and readable.
As an imperfect metric we use the [Go report card](https://goreportcard.com/report/github.com/vs-ude/fyrlang) to find places that need improving.

To ensure consistent indentation and encoding, we use the editorconfig framework. Its settings are stored in the [.editorconfig](./.editorconfig) file.
Many applications support it natively. Please refer to the [documentation](https://editorconfig.org/#download) on how to enable or install it.


### API Documentation

The internal API documentation is automatically built and hosted on [GoDoc](https://godoc.org/github.com/vs-ude/fyrlang).
It updates periodically.
If it is out of date, you can use the link at the bottom of the page to refresh it.


### Project Structure

The project structure tries to follow the layout suggested by the [GoLang Standards](https://github.com/golang-standards/project-layout) project.
A quick overview:

```
.vscode/         | Configuration for Visual Studio Code.
build/           | Configuration files for CI and packaging.
cmd/             | The commands that should be compiled into binaries. One per folder.
examples/        | Examples of Fyr modules. These are used to test the compiler.
internal/        | This folder contains the Go modules that make up the compiler.
lib/             | The standard library of Fyr. Written in Fyr and supported backend languages.
test/            | Additional resources required for testing.
Makefile         | The build and test process definitions used by make.
```


### Building

We use _make_ to manage the build process.
To build the compiler binary, simply type `make fyrc`, executing `make` will build all commands defined in the _cmd/_ folder.
This file also defines actions for testing (`make test`, `make test_go`, `make test_fyr`) and removing generated files (`make clean`) in the project.

More commands may be available.
Please refer to the [Makefile](./Makefile) in the source folder for these.


### Testing

We use two kinds of tests, both of which are integrated into our [CI](https://travis-ci.org/vs-ude/fyrlang) pipeline.
To run all implemented tests you can simply call `make test`.

#### High-level

To test the whole compiler, we have a simple [script](./test/fyr_code_tests.sh) that tries to compile some test files and run the resulting binaries.
The Fyr modules used to test are expected to be located in the _examples/_ directory.  
The script only depends on `/bin/bash` and `date` so it should run on most systems.
It naively checks the exit codes of the compiler and the binaries and outputs files for which it was not `0`.  
To check for possible memory leaks we use [valgrind](http://valgrind.org/).
The script works without it but will complain about the missing dependency.  
You can invoke it with `make test_fyr`.

#### Integrated Tests

We utilize the integrated [Go testing framework](https://golang.org/pkg/testing/) for internal testing of the compiler.
The tests can be run with `make test_go`.
Please refer to the official documentation in order to learn how to write tests in Go.
