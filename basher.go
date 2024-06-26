// Package basher provides an API for running and integrating with Bash from Go
package basher

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/kardianos/osext"
	"github.com/mitchellh/go-homedir"
)

func exitStatus(err error) (int, error) {
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// There is no platform independent way to retrieve
			// the exit code, but the following will work on Unix
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() != -1 {
					return status.ExitStatus(), err
				} else {
					// The process hasn't exited or was terminated by a signal.
					return int(status), err
				}
			}
		}
		return 0, err
	}
	return 0, nil
}

// Application sets up a common entrypoint for a Bash application that
// uses exported Go functions. It uses the DEBUG environment variable
// to set debug on the Context, and SHELL for the Bash binary if it
// includes the string "bash". You can pass a loader function to use
// for the sourced files, and a boolean for whether or not the
// environment should be copied into the Context process.
func Application(
	funcs map[string]func([]string),
	scripts []string,
	loader func(string) ([]byte, error),
	copyEnv bool) {

	bashDir, err := homedir.Expand("~/.basher")
	if err != nil {
		log.Fatal(err, "1")
	}

	bashPath := bashDir + "/bash"
	if _, err := os.Stat(bashPath); os.IsNotExist(err) {
		err = RestoreAsset(bashDir, "bash")
		if err != nil {
			log.Fatal(err, "1")
		}
	}

	ApplicationWithPath(funcs, scripts, loader, copyEnv, bashPath)
}

// ApplicationWithPath functions as Application does while also
// allowing the developer to modify the specified bashPath.
func ApplicationWithPath(
	funcs map[string]func([]string),
	scripts []string,
	loader func(string) ([]byte, error),
	copyEnv bool,
	bashPath string) {

	bash, err := NewContext(bashPath, os.Getenv("DEBUG") != "")
	if err != nil {
		log.Fatal(err)
	}
	for name, fn := range funcs {
		bash.ExportFunc(name, fn)
	}
	if bash.HandleFuncs(os.Args) {
		os.Exit(0)
	}

	for _, script := range scripts {
		if err := bash.Source(script, loader); err != nil {
			log.Fatal(err)
		}
	}
	if copyEnv {
		bash.CopyEnv()
	}
	status, err := bash.Run("main", os.Args[1:])
	if err != nil {
		// the string message for ExitError shouldn't be logged
		// as it is just `exit status $CODE`, which is redundant
		// when that code can just be used to exit the program
		if _, ok := err.(*exec.ExitError); ok && strings.HasPrefix(err.Error(), "exit status ") {
			os.Exit(status)
		} else {
			log.Fatal(err)
		}
	}
	os.Exit(status)
}

// A Context is an instance of a Bash interpreter and environment, including
// sourced scripts, environment variables, and embedded Go functions
type Context struct {
	sync.Mutex

	// Debug simply leaves the generated BASH_ENV file produced
	// from each Run call of this Context for debugging.
	Debug bool

	// BashPath is the path to the Bash executable to be used by Run
	BashPath string

	// SelfPath is set by NewContext to be the current executable path.
	// It's used to call back into the calling Go process to run exported
	// functions.
	SelfPath string

	// The io.Reader given to Bash for STDIN
	Stdin io.Reader

	// The io.Writer given to Bash for STDOUT
	Stdout io.Writer

	// The io.Writer given to Bash for STDERR
	Stderr io.Writer

	vars    []string
	scripts [][]byte
	funcs   map[string]func([]string)
}

// Creates and initializes a new Context that will use the given Bash executable.
// The debug mode will leave the produced temporary BASH_ENV file for inspection.
func NewContext(bashpath string, debug bool) (*Context, error) {
	executable, err := osext.Executable()
	if err != nil {
		return nil, err
	}
	return &Context{
		Debug:    debug,
		BashPath: bashpath,
		SelfPath: executable,
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		scripts:  make([][]byte, 0),
		vars:     make([]string, 0),
		funcs:    make(map[string]func([]string)),
	}, nil
}

// Copies the current environment variables into the Context
func (c *Context) CopyEnv() {
	c.Lock()
	defer c.Unlock()
	c.vars = append(c.vars, os.Environ()...)
}

// Source adds a shell script to the Context environment. The loader argument can be nil
// which means it will use os.Readfile and load from disk, but it exists so you
// can use the Asset function produced by go-bindata when including script files in
// your Go binary. Calls to Source adds files to the environment in order.
func (c *Context) Source(filepath string, loader func(string) ([]byte, error)) error {
	if loader == nil {
		loader = os.ReadFile
	}
	data, err := loader(filepath)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.scripts = append(c.scripts, data)
	return nil
}

// Export adds an environment variable to the Context
func (c *Context) Export(name string, value string) {
	c.Lock()
	defer c.Unlock()
	c.vars = append(c.vars, name+"="+value)
}

// Registers a function with the Context that will produce a Bash function in the environment
// that calls back into your executable triggering the function defined as fn.
func (c *Context) ExportFunc(name string, fn func([]string)) {
	c.Lock()
	defer c.Unlock()
	c.funcs[name] = fn
}

// Expects your os.Args to parse and handle any callbacks to Go functions registered with
// ExportFunc. You normally call this at the beginning of your program. If a registered
// function is found and handled, HandleFuncs will exit with the appropriate exit code for you.
func (c *Context) HandleFuncs(args []string) bool {
	for i, arg := range args {
		if arg == ":::" && len(args) > i+1 {
			c.Lock()
			defer c.Unlock()
			for cmd := range c.funcs {
				if cmd == args[i+1] {
					c.funcs[cmd](args[i+2:])
					return true
				}
			}
			return false
		}
	}
	return false
}

func (c *Context) buildEnvfile() (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "bashenv.")
	if err != nil {
		return "", err
	}
	defer file.Close()
	// variables
	file.Write([]byte("unset BASH_ENV\n")) // unset for future calls to bash
	file.Write([]byte("export SELF=" + os.Args[0] + "\n"))
	file.Write([]byte("export SELF_EXECUTABLE='" + c.SelfPath + "'\n"))
	for _, kvp := range c.vars {
		pair := strings.SplitN(kvp, "=", 2)
		if len(pair) != 2 || strings.TrimSpace(pair[0]) == "" {
			continue
		}

		if isBashFunc(pair[0], pair[1]) {
			bash_function_name := strings.TrimPrefix(pair[0], "BASH_FUNC_")
			bash_function_name = strings.TrimSuffix(bash_function_name, "%%")
			file.Write([]byte(fmt.Sprintf("%s%s\n", bash_function_name, pair[1])))
			file.Write([]byte(fmt.Sprintf("export -f %s\n", bash_function_name)))
			continue
		}

		file.Write([]byte("export " + strings.Replace(
			strings.Replace(kvp, "'", "\\'", -1), "=", "=$'", 1) + "'\n"))
	}
	// functions
	for cmd := range c.funcs {
		file.Write([]byte(cmd + "() { $SELF_EXECUTABLE ::: " + cmd + " \"$@\"; }\n"))
	}
	// scripts
	for _, data := range c.scripts {
		file.Write(append(data, '\n'))
	}
	return file.Name(), nil
}

func isBashFunc(key string, value string) bool {
	return strings.HasPrefix(key, "BASH_FUNC_") && strings.HasPrefix(value, "()")
}

// Runs a command in Bash from this Context. With each call, a temporary file
// is generated used as BASH_ENV when calling Bash that includes all variables,
// sourced scripts, and exported functions from the Context. Standard I/O by
// default is attached to the calling process I/O. You can change this by setting
// the Stdout, Stderr, Stdin variables of the Context.
func (c *Context) Run(command string, args []string) (int, error) {
	c.Lock()
	defer c.Unlock()
	envfile, err := c.buildEnvfile()
	if err != nil {
		return 0, err
	}
	if !c.Debug {
		defer os.Remove(envfile)
	}
	argstring := ""
	for _, arg := range args {
		argstring = argstring + " '" + strings.Replace(arg, "'", "'\\''", -1) + "'"
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	signal.Ignore(syscall.SIGURG)

	cmd := exec.Command(c.BashPath, "-c", command+argstring)
	cmd.Env = []string{"BASH_ENV=" + envfile}
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	errChan := make(chan error, 1)
	go func() {
		for sig := range signals {
			if sig != syscall.SIGCHLD {
				err = cmd.Process.Signal(sig)
				if err != nil {
					errChan <- err
				}
			}
		}
	}()
	go func() {
		errChan <- cmd.Wait()
	}()
	err = <-errChan
	return exitStatus(err)
}
