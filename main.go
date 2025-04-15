package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	j "github.com/pyke369/golang-support/jsonrpc"
)

var (
	extract = "_extract"
	stats   = []int{0, 0, 0}
)

func tree(data []byte, size int, in map[string]any, prefix string) {
	for name := range in {
		if dir := j.Map(in[name])["files"]; dir != nil {
			tree(data, size, j.Map(dir), filepath.Join(prefix, name))
		} else {
			file := j.Map(in[name])
			foffset, fsize, passed := int(j.Number(file["offset"])), int(j.Number(file["size"])), true
			if foffset+fsize <= size {
				integrity := j.Map(file["integrity"])
				if strings.EqualFold(j.String(integrity["algorithm"]), "SHA256") {
					if hash := j.String(integrity["hash"]); len(hash) == 64 {
						passed = hash == fmt.Sprintf("%x", sha256.Sum256(data[foffset:foffset+fsize]))
					}
				}
				if passed {
					if os.Args[1] == "list" {
						fmt.Printf("%12d | %s\n", fsize, filepath.Join(prefix, name))
						stats[0]++
						stats[2] += fsize
					} else {
						path := filepath.Join(extract, prefix, name)
						os.MkdirAll(filepath.Dir(path), 0o755)
						if os.WriteFile(path, data[foffset:foffset+fsize], 0o644) == nil {
							stats[0]++
							stats[2] += fsize
						} else {
							stats[1]++
						}
						fmt.Printf("\r%5d %5d %12d", stats[0], stats[1], stats[2])
					}
				}
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 || (os.Args[1] != "list" && os.Args[1] != "extract") {
		fmt.Fprintf(os.Stderr, "usage: %s list|extract <archive> [<extract>]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	if len(os.Args) > 3 {
		extract = os.Args[3]
	}
	if os.Args[1] == "extract" {
		if _, err := os.Stat(extract); err == nil {
			fmt.Fprintf(os.Stderr, "remove %s first - aborting\n", extract)
			os.Exit(1)
		}
	}

	handle, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s - aborting\n", err.Error())
		os.Exit(2)
	}

	info, _ := handle.Stat()
	size := int(info.Size())
	if size < 16 {
		fmt.Fprintf(os.Stderr, "invalid file - aborting\n")
		os.Exit(3)
	}
	data, err := syscall.Mmap(int(handle.Fd()), 0, (size+4095)&^4095, syscall.PROT_READ, syscall.MAP_SHARED|syscall.MAP_NORESERVE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s aborting\n", err.Error())
		os.Exit(3)
	}
	if binary.LittleEndian.Uint32(data) != 4 {
		fmt.Fprintf(os.Stderr, "invalid header - aborting\n")
		os.Exit(4)
	}

	tsize := int(binary.LittleEndian.Uint32(data[12:]))
	if 16+tsize > size {
		fmt.Fprintf(os.Stderr, "invalid table of content - aborting\n")
		os.Exit(5)
	}
	if tsize == 0 {
		return
	}

	toc := map[string]any{}
	if err := json.Unmarshal(data[16:16+tsize], &toc); err != nil {
		fmt.Fprintf(os.Stderr, "invalid table of content - aborting\n")
		os.Exit(5)
	}
	tree(data[16+tsize:], size-tsize-16, j.Map(toc["files"]), "")
	if os.Args[1] == "extract" {
		fmt.Printf("\rfiles:%d errors:%d size:%d\n", stats[0], stats[1], stats[2])
	} else {
		fmt.Printf("\n%12d | %d files\n", stats[2], stats[0])
	}
}
