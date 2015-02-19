package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"github.com/honr/vulcan/htl"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	tree, err := htl.Parse(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println(tree);
	}
}
