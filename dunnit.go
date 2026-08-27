package main

import (
	"fmt"
	"gsd/dunnit"
)

func main() {
	fmt.Println("Starting Dunzo")

	a := dun.MakeUI()
	w := dun.BuildMainWindow(*a)
	s := dun.Schedule(*a, w)
	defer s.Shutdown()

	(*a).Run()
}
