# go-basher

An API for creating Bash environments, exporting Go functions in them as Bash functions, and running commands in that Bash environment. Combined with a tool like [go-bindata](https://github.com/jteeuwen/go-bindata), you can write programs that are part written in Go and part written in Bash that can be distributed as standalone binaries.

[![Build Status](https://travis-ci.org/progrium/go-basher.png)](https://travis-ci.org/progrium/go-basher) [![GoDoc](https://godoc.org/github.com/progrium/go-basher?status.svg)](http://godoc.org/github.com/progrium/go-basher)


## Motivation

Go is a great compiled systems language, but it can still be faster to write and glue existing commands together in Bash. However, there are operations you wouldn't want to do in Bash that are straightforward in Go, for example, writing and reading structured data formats. By allowing them to work together, you can use each where they are strongest.

Take a common task like making an HTTP request for JSON data. Parsing JSON is easy in Go, but without depending on a tool like `jq` it is not even worth trying in Bash. And some formats like YAML don't even have a good `jq` equivalent. Whereas making an HTTP request in Go in the *simplest* case is going to be 6+ lines, as opposed to Bash where you can use `curl` in one line. If we write our JSON parser in Go and fetch the HTTP doc with `curl`, we can express printing a field from a remote JSON object in one line:

	curl https://api.github.com/users/progrium | parse-user-field email

In this case, the command `parse-user-field` is an app specific function defined in your Go program.

Why would this ever be worth it? I can think of several basic cases:

 1. you're writing a program in Bash that involves some complex functionality that should be in Go
 1. you're writing a CLI tool in Go but, to start, prototyping would be quicker in Bash
 1. you're writing a program in Bash and want it to be easier to distribute, like a Go binary

## License

BSD