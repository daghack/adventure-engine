package main

import (
	"adventure/engine/v2"
)

func main() {
	eng, _ := engine.StartEngine("", "")
	err := eng.LoadStoryPage("first", "stories/test")
	if err != nil {
		panic(err)
	}
	err = eng.RunPage("first")
	if err != nil {
		panic(err)
	}
}
