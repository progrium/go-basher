# go-basher

A Go library for creating Bash environments, exporting Go functions in them as Bash functions, and running commands in that Bash environment. Combined with a tool like [go-bindata](https://github.com/jteeuwen/go-bindata), you can write programs that are part written in Go and part written in Bash that can be distributed as standalone binaries.

[![Circle CI](https://circleci.com/gh/progrium/go-basher.svg?style=shield)](https://circleci.com/gh/progrium/go-basher) [![GoDoc](https://godoc.org/github.com/progrium/go-basher?status.svg)](http://godoc.org/github.com/progrium/go-basher)

## Using go-basher

Here we have a simple Go program that defines a `reverse` function, creates a Bash environment sourcing `main.bash` and then runs `main` in that environment.

```Go
package main

import (
	"os"
	"io/ioutil"
	"log"
	"strings"

	"github.com/progrium/go-basher"
)

func reverse(args []string) {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	runes := []rune(strings.Trim(string(bytes), "\n"))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	println(string(runes))
}

func main() {
	bash, _ := basher.NewContext("/bin/bash", false)
	bash.ExportFunc("reverse", reverse)
	if bash.HandleFuncs(os.Args) {
		os.Exit(0)
	}

	bash.Source("main.bash", nil)
	status, err := bash.Run("main", os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(status)
}
```

Here is our `main.bash` file, the actual heart of the program:

```bash
main() {
	echo "Hello world" | reverse
}
```

## Using go-basher with go-bindata

You can bundle your Bash scripts into your Go binary using [go-bindata](https://github.com/jteeuwen/go-bindata). First install go-bindata:

	$ go get github.com/jteeuwen/go-bindata/...

Now put all your Bash scripts in a directory called `bash`. The above example program would mean you'd have a `bash/main.bash` file. Run `go-bindata` on the directory:

	$ go-bindata bash

This will produce a `bindata.go` file that includes all of your Bash scripts.

> `bindata.go` includes a function called `Asset` that behaves like `ioutil.ReadFile` for files in your `bindata.go`. 

Here's how you embed it into the above example program:

* copy/paste it's import-statements and functions to your application code
* method A: change `bash.Source("bash/main.bash", nil)` into `bash.Source("bash/main.bash, Asset)`
* method B: replace all code in the `main()`-function with the `Application()`-helper function (see below)

```Go
	basher.Application(
		map[string]func([]string){
			"reverse":      reverse,
		}, []string{
			"bash/main.bash",
		},
		Asset,
		true,
	)
```

## Batteries included, but replaceable

Did you already hear that term? Sometimes Bash binary is missing, for example when using alpine linux or busybox. Or sometimes its not the correct version. Like OSX ships with Bash 3.x which misses a lot of usefull features. Or you want to make sure to avoid shellshock attack.

For those reasons static versions of Bash binaries are included for linux and darwin. Statically compiled bash-4.3.30 is released on github: https://github.com/robxu9/bash-static. These are then turned into go code, with go-bindata: bindata_linux.go and bindata_darwin.go.

When you use the `basher.Application()` function, the built in Bash binary will be extracted into the `~/.basher/` dir.

When you use the `basher.NewContext()` function, you have to specify the path to Bash.

## Motivation

Go is a great compiled systems language, but it can still be faster to write and glue existing commands together in Bash. However, there are operations you wouldn't want to do in Bash that are straightforward in Go, for example, writing and reading structured data formats. By allowing them to work together, you can use each where they are strongest.

Take a common task like making an HTTP request for JSON data. Parsing JSON is easy in Go, but without depending on a tool like `jq` it is not even worth trying in Bash. And some formats like YAML don't even have a good `jq` equivalent. Whereas making an HTTP request in Go in the *simplest* case is going to be 6+ lines, as opposed to Bash where you can use `curl` in one line. If we write our JSON parser in Go and fetch the HTTP doc with `curl`, we can express printing a field from a remote JSON object in one line:

	curl -s https://api.github.com/users/progrium | parse-user-field email

In this case, the command `parse-user-field` is an app specific function defined in your Go program.

Why would this ever be worth it? I can think of several basic cases:

 1. you're writing a program in Bash that involves some complex functionality that should be in Go
 1. you're writing a CLI tool in Go but, to start, prototyping would be quicker in Bash
 1. you're writing a program in Bash and want it to be easier to distribute, like a Go binary

## License

BSD
