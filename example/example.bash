
set -eo pipefail

hello-bash() {
	echo "Hello world from Bash"
}

main() {
	echo "Arguments:" "$@"
	hello-bash
	hello-go
	hello-bash | reverse
	curl -s https://api.github.com/repos/progrium/go-basher | jpointer /owner/login
}
