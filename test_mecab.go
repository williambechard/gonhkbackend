package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	input := "テスト文章"
	cmd := exec.Command("mecab")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	// Print each token (surface form)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "EOS" || line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) > 0 {
			fmt.Println(fields[0])
		}
	}
}
