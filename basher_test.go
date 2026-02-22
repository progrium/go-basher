package basher

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"testing"
)

var bashpath = "/bin/bash"

var testScripts = map[string]string{
	"hello.sh":  `main() { echo "hello"; }`,
	"cat.sh":    `main() { cat; }`,
	"printf.sh": `main() { printf "arg: <%s>" "$@"; }`,
	"foobar.sh": `main() { echo $FOOBAR; }`,
}

func testLoader(name string) ([]byte, error) {
	return []byte(testScripts[name]), nil
}

func TestHelloStdout(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("hello.sh", testLoader)

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("main", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit")
	}
	if stdout.String() != "hello\n" {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestHelloStdin(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("cat.sh", testLoader)
	bash.Stdin = bytes.NewBufferString("hello\n")

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("main", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit")
	}
	if stdout.String() != "hello\n" {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestEnvironment(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	complexString := "Andy's Laptop says, \"$X=1\""
	bash.Source("foobar.sh", testLoader)
	bash.Export("FOOBAR", complexString)

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("main", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit")
	}
	if strings.Trim(stdout.String(), "\n") != complexString {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestFuncCallback(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.ExportFunc("myfunc", func(args []string) {
		return
	})
	bash.SelfPath = "/bin/echo"

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("myfunc", []string{"abc", "123"})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit")
	}
	if stdout.String() != "::: myfunc abc 123\n" {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestFuncHandling(t *testing.T) {
	exit := make(chan int, 1)
	bash, _ := NewContext(bashpath, false)
	bash.ExportFunc("test-success", func(args []string) {
		exit <- 0
	})
	bash.ExportFunc("test-fail", func(args []string) {
		exit <- 2
	})

	bash.HandleFuncs([]string{"thisprogram", ":::", "test-success"})
	status := <-exit
	if status != 0 {
		t.Fatal("non-zero exit")
	}

	bash.HandleFuncs([]string{"thisprogram", ":::", "test-fail"})
	status = <-exit
	if status != 2 {
		t.Fatal("unexpected exit status:", status)
	}
}

func TestOddArgs(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("printf.sh", testLoader)

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("main", []string{"hel\n\\'lo"})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit")
	}

	if stdout.String() != "arg: <hel\n\\'lo>" {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestRunWithSocketStdin(t *testing.T) {
	// Bash 5.x does not source BASH_ENV when stdin is a Unix socket.
	// This is the default configuration in go-basher since NewContext
	// sets Stdin to os.Stdin, which may be a socket (e.g. when the
	// calling process is managed by a process supervisor, systemd
	// socket activation, or similar).
	//
	// The fix is to explicitly source the envfile in the -c command
	// string instead of relying on BASH_ENV.
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Skip("socketpair not available:", err)
	}
	sockFile := os.NewFile(uintptr(fds[0]), "socket")
	defer sockFile.Close()
	defer syscall.Close(fds[1])

	bash, _ := NewContext(bashpath, false)
	bash.Source("hello.sh", testLoader)
	bash.Stdin = sockFile

	var stdout bytes.Buffer
	bash.Stdout = &stdout
	status, err := bash.Run("main", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if status != 0 {
		t.Fatal("non-zero exit, status:", status)
	}
	if stdout.String() != "hello\n" {
		t.Fatal("unexpected stdout:", stdout.String())
	}
}

func TestIsBashFunc(t *testing.T) {
	if isBashFunc("", "") {
		t.Fatal("empty string is not a bash func")
	}

	if isBashFunc("key", "value") {
		t.Fatal("key=value is not a bash func")
	}

	if isBashFunc("BASH_FUNC_readlinkf", "value") {
		t.Fatal("key does not end with %%")
	}

	if isBashFunc("BASH_FUNC_readlinkf%%", "value") {
		t.Fatal("value does not begin with ()")
	}
	if !isBashFunc("BASH_FUNC_readlinkf%%", "() { true }") {
		t.Fatal("bash func should be detected")
	}
}
