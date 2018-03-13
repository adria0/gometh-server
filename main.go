package main

import (
	gometh "github.com/adriamb/gometh-server/gometh"

	"github.com/CrowdSurge/banner"
)

func main() {
	banner.Print("gometh")
	gometh.ExecuteCmd()
}
