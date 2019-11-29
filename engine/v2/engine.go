package engine

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/aarzilli/golua/lua"
	"github.com/stevedonovan/luar"
)

// Engine Config File
const example_config = `
base_lua = "base.lua"
story_dir = "stories"
`

// Story File
const example_story = `
local config = {
	time_on_screen_ms = 2000,
	transition = "fade",
	transition_time_ms = 500
}

local story = {
	"The world fades as $[ name_of_stranger ] pulls you into darkness.",
	"Moments.",
	"Hours.",
	"Weeks.",
	"Years.",
	"It is difficult to say."
}

local actions = {
	reload: {
		text: "Reload this same story page.",
	},
	hold_hand: {
		text: "Continue holding on to $[ name_of_stranger ]'s hand.",
		transition_to: "first"
	},
	let_go: {
		text: "Let go of $[ name_of_stranger ]'s hand.",
		execute: "into_the_darkness = true",
		transition_to: "second"
	},
	light_candle: {
		cond: "inventory.candle",
		text: "Lift your candle into the darkness.",
		transition_to: "third"
	}
}
`

func StartEngine(configFile, storyName string) (*Engine, error) {
	return &Engine{
		state: luar.Init(),
	}, nil
}

func (eng *Engine) LoadStoryPage(page, storydir string) error {
	pagefile := filepath.Join(storydir, page+".page")
	res := eng.state.LoadFile(pagefile)
	if res != 0 {
		msg := eng.state.ToString(-1)
		eng.state.Pop(1)
		return fmt.Errorf("Failed to load story file [%s]: %s", pagefile, msg)
	}
	eng.state.NewTable()
	eng.state.SetGlobal(page)
	eng.state.GetGlobal(page)
	eng.state.SetfEnv(-2)
	err := eng.state.Call(0, 0)
	if err != nil {
		return err
	}
	eng.state.GetGlobal(page)
	pageValues := map[string]interface{}{}
	err = luar.LuaToGo(eng.state, -1, &pageValues)
	if err != nil {
		return err
	}
	fmt.Printf("Loaded [%s]: %+v\n", page, pageValues)
	return nil
}

type Engine struct {
	state *lua.State
	pages map[string]*StoryPage
}

type StoryPage struct {
	Name    string
	Story   *Story
	Actions *ActionSet
	Config  *PageConfig
}

type PageConfig struct {
	TimeOnScreen   time.Duration
	TransitionType string
	TransitionTime time.Duration
}

type Story struct {
	Sections []string
}

type ActionSet struct {
	Actions []*Action
}

type Action struct {
	Text         string
	Cond         *string
	TransitionTo *string
	Execute      *string
}
