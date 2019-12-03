package main

import (
	"adventure/engine/v2"
	"fmt"
)

func main() {
	eng, err := engine.StartEngine("./engine_conf.lua", "test")
	if err != nil {
		panic(err)
	}
	err = eng.LoadStoryPage("first", "stories/test")
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
	actions, err := eng.RenderActions()
	if err != nil {
		panic(err)
	}
	for _, action := range actions {
		fmt.Println(action.ActionStr, action.RenderedText)
	}
}
