package engine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
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
		pages: map[string]*StoryPage{},
	}, nil
}

func runReturnLuaString(state *lua.State, str string) (string, error) {
	t := state.GetTop()
	err := state.DoString("return " + str)
	if err != nil {
		return "", err
	}
	t_p := state.GetTop()
	if t == t_p {
		return "", nil
	}
	toret := state.ToString(-1)
	state.Pop(t_p - t)
	return toret, nil
}

type Engine struct {
	state       *lua.State
	pages       map[string]*StoryPage
	currentPage string
}

func (eng *Engine) LoadStoryPage(page, storydir string) error {
	baseLua := filepath.Join(storydir, "base.lua")
	err := eng.state.DoFile(baseLua)
	if err != nil {
		return err
	}
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
	err = eng.state.Call(0, 0)
	if err != nil {
		return err
	}
	eng.state.GetGlobal(page)
	defer eng.state.Pop(1)
	pageValues := map[string]interface{}{}
	err = luar.LuaToGo(eng.state, -1, &pageValues)
	if err != nil {
		return err
	}
	eng.pages[page] = &StoryPage{
		Story:   eng.buildStory(pageValues["story"]),
		Actions: eng.buildActions(pageValues["actions"]),
		Config:  eng.buildConfig(pageValues["config"]),
	}
	fmt.Printf("Loaded [%s]:\n\t%+v\n", page, eng.pages[page])
	return nil
}

func (eng *Engine) RunPage(page string) error {
	p, ok := eng.pages[page]
	if !ok {
		return fmt.Errorf("No such page [%s]", page)
	}
	sections, err := p.Story.RenderSections(eng.state)
	if err != nil {
		return err
	}
	for _, section := range sections {
		fmt.Println(section)
	}
	return nil
}

func (eng *Engine) buildStory(story interface{}) *Story {
	storySections := story.([]interface{})
	section_strs := []string{}
	for _, section := range storySections {
		sect_str := section.(string)
		section_strs = append(section_strs, sect_str)
	}
	return &Story{
		Sections: section_strs,
	}
}

func (eng *Engine) buildActions(actions interface{}) *ActionSet {
	actionsInterface := actions.(map[string]interface{})
	toret := &ActionSet{
		Actions: map[string]*Action{},
	}
	for actionName, actionInterface := range actionsInterface {
		action := actionInterface.(map[string]interface{})
		text := action["text"].(string)
		toret.Actions[actionName] = &Action{
			Text: text,
		}
		transition_to, ok := action["transition_to"]
		if ok {
			transition_to_str := transition_to.(string)
			toret.Actions[actionName].TransitionTo = &transition_to_str
		}
		execute, ok := action["execute"]
		if ok {
			execute_str := execute.(string)
			toret.Actions[actionName].Execute = &execute_str
		}
		cond, ok := action["cond"]
		if ok {
			cond_str := cond.(string)
			toret.Actions[actionName].Cond = &cond_str
		}
	}
	return toret
}

func (eng *Engine) buildConfig(config interface{}) *PageConfig {
	confmap := config.(map[string]interface{})
	time_on_screen_float := confmap["time_on_screen_ms"].(float64)
	transition_time_float := confmap["transition_time_ms"].(float64)
	transition := confmap["transition"].(string)
	return &PageConfig{
		TimeOnScreen:   time.Duration(time_on_screen_float) * time.Millisecond,
		TransitionType: transition,
		TransitionTime: time.Duration(transition_time_float) * time.Millisecond,
	}
}

type StoryPage struct {
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

func (story *Story) RenderSections(state *lua.State) ([]string, error) {
	prefix := regexp.QuoteMeta("$[")
	suffix := regexp.QuoteMeta("]")
	r, err := regexp.Compile(prefix + `(?s)(.+?)` + suffix)
	if err != nil {
		return nil, err
	}
	for i, section := range story.Sections {
		matches := r.FindAllStringSubmatch(section, -1)
		for _, match := range matches {
			str, err := runReturnLuaString(state, match[1])
			if err != nil {
				return nil, err
			}
			story.Sections[i] = strings.Replace(story.Sections[i], match[0], str, 1)
		}
	}
	return story.Sections, nil
}

type ActionSet struct {
	Actions map[string]*Action
}

type Action struct {
	Text         string
	Cond         *string
	TransitionTo *string
	Execute      *string
}
