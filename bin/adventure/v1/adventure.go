package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"adventure/engine/v1"
)

func main() {
	adv, err := engine.NewEngine("./stories/test", "./lua")
	if err != nil {
		panic(err)
	}
	actions, story := adv.Run()
	reader := bufio.NewReader(os.Stdin)
	for latest := range story {
		fmt.Println("______________________________")
		fmt.Println(string(latest))
		fmt.Printf("> ")
		action, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		action = strings.TrimSpace(action)
		actions <- engine.Action(action)
	}
}
