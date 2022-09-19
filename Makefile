gazelle:
	bzl run //release:gazelle

refresh:
	bzl build //cmd/ocitool
	cp $(bzl info bazel-bin)/cmd/ocitool/ocitool_/ocitool $(bzl info workspace)/bin/ocitool-linux-amd64
