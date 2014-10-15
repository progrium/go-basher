package main

import (
	"log"
	"os"
	"strings"

	"github.com/progrium/go-basher"
)

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func helloworld(args []string) int {
	print("Hello world from Go\n")
	return 0
}

func echo(args []string) int {
	if len(args) > 0 {
		for _, arg := range args {
			println(arg)
		}
	}
	return 0
}

func reverse(args []string) int {
	bytes, err := ioutil.ReadAll(os.Stdin)
	assert(err)
	runes := []rune(strings.Trim(string(bytes), "\n"))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	println(string(runes))
	return 0
}

func main() {
	bash := basher.NewContext()
	bash.ExportFunc("helloworld", helloworld)
	bash.ExportFunc("go-echo", echo)
	bash.ExportFunc("reverse", reverse)
	bash.HandleFuncs()

	bash.Source("./example.bash")
	status, err := bash.Run("main", os.Args)
	assert(err)
	os.Exit(status)
}
