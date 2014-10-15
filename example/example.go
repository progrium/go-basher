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

func helloworld(args []string) {
	print("Hello world from Go\n")
}

func echo(args []string) {
	if len(args) > 0 {
		for _, arg := range args {
			println(arg)
		}
	}
}

func reverse(args []string) {
	bytes, err := ioutil.ReadAll(os.Stdin)
	assert(err)
	runes := []rune(strings.Trim(string(bytes), "\n"))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	println(string(runes))
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
