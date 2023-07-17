package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func runCmd(cmd []string) error {
	var buffer bytes.Buffer
	writer := io.MultiWriter(os.Stdout, &buffer)
	command := exec.Command(cmd[0], cmd[1:]...)
	command.Stdout = writer

	return command.Run()
}

func startServer(binary, wordListPath string, port int) error {
	cmd := []string{
		binary,
		"--word_list_path",
		wordListPath,
		"--port",
		fmt.Sprintf("%d", port),
	}

	return runCmd(cmd)
}

func main() {
	go_port := flag.Int("go_port", 6500, "Port for Go server to listen on.")
	cc_port := flag.Int("cc_port", 6501, "Port for C++ server to listen on.")
	wordListPath := flag.String("word_list_path", "", "List of candidate words. One word per line.")
	workspaceRoot := flag.String("workspace_root", "", "Bazel root workspace path.")
	flag.Parse()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		binary := filepath.Join(*workspaceRoot, "bazel-bin/wordle_go_/wordle_go")
		if err := startServer(binary, *wordListPath, *go_port); err != nil {
			log.Fatalln(err)
		}
	}()

	go func() {
		binary := filepath.Join(*workspaceRoot, "bazel-bin/wordle_cc")
		if err := startServer(binary, *wordListPath, *cc_port); err != nil {
			log.Fatalln(err)
		}
	}()

	wg.Wait()
}
