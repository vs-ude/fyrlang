# Fyrlang

Fyr is a modern systems programming language that combines the versatility of C with the ease and safety of application programming languages like Java, Go or TypeScript. Like C/C++, Fyr can be used for low-level hardware-oriented programming and high-level application programming. In contrast to C, the Fyr compiler guarantees memory safety and thread safety at compilation time.  

Fyr is designed to implement all tiers of distributed IoT applications, i.e. embedded devices, server-side code and the Web UI. Furthermore, it can be combined with existing C and JavaScript code.  

This is a complete rewrite using Go as a base. We found that the original design was lacking in some aspects, which would have required architectural changes. This made a rewrite an easier solution.
Using Go as a base gives us more freedom in the implementation and improves performance drastically.

Please do not use this version as of now, as it is very much unfinished.

## Development

Setting this project up for development is simple.
Since it does not have any external dependencies you only need to clone this repository and link it into your `$GOPATH`.
_Notice:_ This project requires Go 1.13 or later to work.

To make imports and compilation work, the link needs to look like this:
```
$GOPATH/src/github.com/vs-ude/fyrlang -> /path/to/source
```

To build the compiler run
```
make
```

Now set the `$FYRBASE` environment variable and add the compiler to your path as follows
```
export FYRBASE=$GOPATH/src/github.com/vs-ude/fyrlang
export PATH=$PATH:$FYRBASE
```

For more information on how to contribute please refer to the [contribution guidelines](./CONTRIBUTING.md)

[![Build Status](https://travis-ci.org/vs-ude/fyrlang.svg?branch=dev)](https://travis-ci.org/vs-ude/fyrlang)
[![Go Report Card](https://goreportcard.com/badge/github.com/vs-ude/fyrlang)](https://goreportcard.com/report/github.com/vs-ude/fyrlang)
