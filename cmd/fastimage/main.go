package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/XGFan/fastimage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <file>\n", filepath.Base(os.Args[0]))
		return
	}

	name := os.Args[1]
	file, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file error: %+v", err)
		os.Exit(1)
	}
	defer file.Close()

	info := fastimage.GetInfoReader(file)
	if info.Type == fastimage.Unknown {
		os.Exit(1)
	}

	fmt.Printf("%s %s %d %d\n", info.Type, info.Type.Mime(), info.Width, info.Height)
}
