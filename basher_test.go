package basher

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
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

func TestRunDoesNotLeakGoroutines(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("hello.sh", testLoader)
	bash.Stdout = io.Discard

	if _, err := bash.Run("main", []string{}); err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	baseline := runtime.NumGoroutine()

	const iterations = 50
	for i := 0; i < iterations; i++ {
		status, err := bash.Run("main", []string{})
		if err != nil {
			t.Fatal(err)
		}
		if status != 0 {
			t.Fatalf("non-zero exit on iteration %d", i)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	var current int
	for {
		runtime.GC()
		current = runtime.NumGoroutine()
		if current-baseline <= 5 || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if delta := current - baseline; delta > 5 {
		t.Fatalf("goroutine leak: baseline=%d after %d Run calls=%d (delta=%d)", baseline, iterations, current, delta)
	}
}

type shortWriter struct {
	remaining int
	err       error
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, w.err
	}
	if len(p) <= w.remaining {
		w.remaining -= len(p)
		return len(p), nil
	}
	n := w.remaining
	w.remaining = 0
	return n, w.err
}

func TestWriteEnvfileContents(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.SelfPath = "/bin/echo"
	bash.Source("hello.sh", testLoader)
	bash.Export("FOOBAR", "baz")
	bash.ExportFunc("myfunc", func([]string) {})
	bash.vars = append(bash.vars, "BASH_FUNC_helper%%=() { echo hi; }")

	var buf bytes.Buffer
	if err := bash.writeEnvfile(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	wants := []string{
		"unset BASH_ENV\n",
		"export SELF=",
		"export SELF_EXECUTABLE='/bin/echo'\n",
		"export FOOBAR=$'baz'\n",
		"helper() { echo hi; }\n",
		"export -f helper\n",
		"myfunc() { $SELF_EXECUTABLE ::: myfunc \"$@\"; }\n",
		`main() { echo "hello"; }` + "\n",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("envfile missing %q\nfull output:\n%s", w, out)
		}
	}
}

func TestWriteEnvfileWriteError(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("hello.sh", testLoader)
	bash.Export("FOOBAR", "baz")

	w := &shortWriter{remaining: 8, err: io.ErrShortWrite}
	err := bash.writeEnvfile(w)
	if err == nil {
		t.Fatal("expected error from writeEnvfile when underlying writer fails")
	}
}

func TestBuildEnvfileWritesAndClosesFile(t *testing.T) {
	bash, _ := NewContext(bashpath, false)
	bash.Source("hello.sh", testLoader)
	bash.Export("FOOBAR", "baz")

	name, err := bash.buildEnvfile()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)

	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("envfile is empty")
	}
	if !strings.Contains(string(data), "unset BASH_ENV\n") {
		t.Fatalf("envfile missing sentinel; contents:\n%s", data)
	}
	if !strings.Contains(string(data), `main() { echo "hello"; }`) {
		t.Fatalf("envfile missing sourced script; contents:\n%s", data)
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
