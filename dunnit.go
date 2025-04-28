package main

import (
	"fmt"
	"gsd/dunnit"
	"time"
)

func main() {
	fmt.Println("In dunnit main")

	a := dun.MakeUI()
	s := dun.Schedule(*a)
	dun.StartUI(*a)


	// inspect(c.Entries())
	dun.SetThings()
	// var f Fooey; f.DoFoo()
	// fmt.Println(Bar())

	select { // block until you are ready to shut down
	case <-time.After(4 * time.Second):
		fmt.Println("a minute has passed")
	}

	err := s.Shutdown() // when you're done, shut it down
	if err != nil {}

}
