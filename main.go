package main

import (
	"fmt"
	"os"

	"github.com/fengxsong/exporter-care/cmd"
)

const appName = "exporter-care"

func main() {
	if err := cmd.NewCommand(appName).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
