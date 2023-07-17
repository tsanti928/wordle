# Wordle Suggester

Assumes `Bazel` and `npm` are installed.

The provided wordlist is a copy of https://gist.github.com/scholtes/94f3c0303ba6a7768b47583aff36654d#file-wordle-la-txt.

## Build Go Server

`bazel build wordle_go`

## Build C++ Server

`bazel build wordle_cc`

## Run Both Backends

From the git root:

`bazel run both -- --workspace_root=$PWD --word_list_path=$PWD/wordlist.txt`

## Launch Frontend

Assumes `npm live-server` is installed.

`live-server .`

## Known Issues

Access to the C++ backend from the browser suffers from CORS issues. CURL or Postman are able to get around this limitation.