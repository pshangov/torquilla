package main

import (
	"github.com/pshangov/torquilla/tq"
	"os"
)

func main() {
	if err := tq.RootCmd.Execute(); err != nil {
		//fmt.Println(err)
		os.Exit(-1)
	}
}
