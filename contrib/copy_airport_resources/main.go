package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Missing source directory")
		os.Exit(1)
	}
	path := os.Args[1]
	s, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		fmt.Println("airports directory does not exist")
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("error checking file status: %s\n", err.Error())
		os.Exit(1)
	} else if !s.IsDir() {
		fmt.Println("airports path is not a directory (is it a file?)")
		os.Exit(1)
	}

	err = filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if info.IsDir() {
			return nil
		}
		if len(fp) > 4 && fp[len(fp)-4:] == ".aip" {
			dat, err := ioutil.ReadFile(fp)
			if err != nil {
				return err
			}
			ds := strings.Split(fp, "/")
			return ioutil.WriteFile("./build/airports/"+ds[len(ds)-1], dat, 0664)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("error processing files: %s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
