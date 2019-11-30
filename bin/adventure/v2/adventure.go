package main

import (
	"adventure/engine/v2"
	"fmt"
)

func main() {
	eng, _ := engine.StartEngine("", "stories/test")
	err := eng.LoadStoryPage("first", "stories/test")
	if err != nil {
		panic(err)
	}
	err = eng.RunPage("first")
	if err != nil {
		panic(err)
	}
	err = eng.RunAction("reload")
	fmt.Println(err)
	err = eng.RunAction("hold_hand")
	fmt.Println(err)
	err = eng.RunAction("let_go")
	fmt.Println(err)
	err = eng.RunAction("light_candle")
	fmt.Println(err)
}
