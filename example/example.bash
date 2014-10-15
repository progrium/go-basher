
set -eo pipefail

bash-helloworld() {
	echo "Hello world from Bash"
}

callgo() {
	helloworld
	go-echo "$@"
	cat | reverse
}

main() {
	echo $1
	bash-helloworld
	callgo "$@"
}
