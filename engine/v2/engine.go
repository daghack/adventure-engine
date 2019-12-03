package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

func runReturnLuaString(state *lua.State, page, str string) (string, error) {
	t := state.GetTop()
	err := state.DoString("return " + str)
	t_p := state.GetTop()
	defer state.Pop(t_p - t)
	if err != nil {
		return "", err
	}
	if t == t_p {
		return "", nil
	}
	toret := state.ToString(-1)
	return toret, nil
}

func runReturnLuaBool(state *lua.State, str string) (bool, error) {
	t := state.GetTop()
	err := state.DoString("return " + str)
	t_p := state.GetTop()
	defer state.Pop(t_p - t)
	if err != nil {
		return false, err
	}
	if t == t_p {
		return false, nil
	}
	toret := state.ToBoolean(-1)
	return toret, nil
}

type Engine struct {
	state       *lua.State
	pages       map[string]*StoryPage
	storyDir    string
	htmlDir     string
	luaDir      string
	currentPage string
	story       string
	exprRegex   *regexp.Regexp
}

func StartEngine(configFile, story string) (*Engine, error) {
	eng := &Engine{
		state: luar.Init(),
		pages: map[string]*StoryPage{},
		story: story,
	}
	err := eng.loadConfig(configFile)
	if err != nil {
		return nil, err
	}
	err = eng.loadLua(eng.luaDir, "", "")
	if err != nil {
		return nil, err
	}
	storylua := filepath.Join(eng.storyDir, story, "lua")
	err = eng.loadLua(storylua, story, "engine")
	if err != nil {
		return nil, err
	}
	prefix := regexp.QuoteMeta("$[")
	suffix := regexp.QuoteMeta("]")
	r, err := regexp.Compile(prefix + `(?s)(.+?)` + suffix)
	if err != nil {
		return nil, err
	}
	eng.exprRegex = r
	return eng, nil
}

func (eng *Engine) loadConfig(configFile string) error {
	confState := luar.Init()
	defer confState.Close()
	err := confState.DoFile(configFile)
	if err != nil {
		return err
	}
	confState.GetGlobal("story_dir")
	eng.storyDir = confState.ToString(-1)
	confState.GetGlobal("lua_dir")
	eng.luaDir = confState.ToString(-1)
	confState.GetGlobal("html_dir")
	eng.htmlDir = confState.ToString(-1)
	return nil
}

func (eng *Engine) loadLua(dir, envName, parentName string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".lua") {
			if envName == "" {
				err = eng.state.DoFile(path)
				if err != nil {
					return err
				}
			} else {
				eng.loadFileInEnv(envName, parentName, path)
			}
			fmt.Printf("Loaded Lua File: [%s]\n", info.Name())
			return nil
		}
		return nil
	})
}

func environmentStr(envName string) string {
	return "environment_" + envName
}

func (eng *Engine) envTableExists(envName string) bool {
	eng.state.GetGlobal(envName)
	defer eng.state.Pop(1)
	return !eng.state.IsNil(-1)
}

func (eng *Engine) createEnvTable(envName, parentName string) {
	eng.state.NewTable()
	eng.state.SetGlobal(envName)
	eng.state.GetGlobal(envName)
	eng.state.GetGlobal("_G")
	eng.state.SetField(-2, parentName)
}

func (eng *Engine) getEnvTable(envName, parentName string) {
	env := environmentStr(envName)
	fmt.Println("Loading:", env)
	if !eng.envTableExists(env) {
		fmt.Println("CREATING ENV:", env)
		eng.createEnvTable(env, parentName)
	} else {
		fmt.Println("LOADING ENV:", env)
		eng.state.GetGlobal(env)
	}
}

func (eng *Engine) runStrInEnvBool(envName, parentName, luacode string) (bool, error) {
	t := eng.state.GetTop()
	res := eng.state.LoadString("return " + luacode)
	if res != 0 {
		msg := eng.state.ToString(-1)
		eng.state.Pop(1)
		return false, fmt.Errorf("Failed to load lua code [%s] into %s env: %s", luacode, envName, msg)
	}
	eng.getEnvTable(envName, parentName)
	eng.state.SetfEnv(-2)
	err := eng.state.Call(0, 1)
	if err != nil {
		return false, err
	}
	t_p := eng.state.GetTop()
	defer eng.state.Pop(t_p - t)
	return eng.state.ToBoolean(-1), nil
}

func (eng *Engine) runStrInEnvStr(envName, parentName, luacode string) (string, error) {
	t := eng.state.GetTop()
	res := eng.state.LoadString("return " + luacode)
	if res != 0 {
		msg := eng.state.ToString(-1)
		eng.state.Pop(1)
		return "", fmt.Errorf("Failed to load lua code [%s] into %s env: %s", luacode, envName, msg)
	}
	eng.getEnvTable(envName, parentName)
	eng.state.SetfEnv(-2)
	err := eng.state.Call(0, 1)
	if err != nil {
		return "", err
	}
	t_p := eng.state.GetTop()
	defer eng.state.Pop(t_p - t)
	return eng.state.ToString(-1), nil
}

func (eng *Engine) runStrInEnv(envName, parentName, luacode string) error {
	t := eng.state.GetTop()
	res := eng.state.LoadString(luacode)
	if res != 0 {
		msg := eng.state.ToString(-1)
		eng.state.Pop(1)
		return fmt.Errorf("Failed to load lua code [%s] into %s env: %s", luacode, envName, msg)
	}
	eng.getEnvTable(envName, parentName)
	eng.state.SetfEnv(-2)
	err := eng.state.Call(0, 0)
	if err != nil {
		return err
	}
	t_p := eng.state.GetTop()
	defer eng.state.Pop(t_p - t)
	return nil
}

func (eng *Engine) loadFileInEnv(envName, parentName, luafile string) error {
	res := eng.state.LoadFile(luafile)
	if res != 0 {
		msg := eng.state.ToString(-1)
		eng.state.Pop(1)
		return fmt.Errorf("Failed to load lua file [%s] into %s env: %s", luafile, envName, msg)
	}
	eng.getEnvTable(envName, parentName)
	eng.state.SetfEnv(-2)
	return eng.state.Call(0, 0)
}

func (eng *Engine) LoadStoryPage(page, storydir string) error {
	eng.currentPage = page
	pagefile := filepath.Join(storydir, page+".page")
	err := eng.loadFileInEnv(page, "engine", pagefile)
	if err != nil {
		return err
	}

	story := []interface{}{}
	actions := map[string]interface{}{}
	config := map[string]interface{}{}

	luaPage := luar.NewLuaObjectFromName(eng.state, environmentStr(page))
	defer luaPage.Close()

	err = luaPage.Get(&config, "config")
	if err != nil {
		return err
	}

	err = luaPage.Get(&story, "story")
	if err != nil {
		return err
	}

	err = luaPage.Get(&actions, "actions")
	if err != nil {
		return err
	}

	eng.pages[page] = &StoryPage{
		Story:   eng.buildStory(story),
		Actions: eng.buildActions(actions),
		Config:  eng.buildConfig(config),
	}

	fmt.Printf("Loaded [%s]:\n\t%+v\n", page, eng.pages[page])
	return nil
}

func (eng *Engine) RunPage(page string) error {
	p, ok := eng.pages[page]
	if !ok {
		return fmt.Errorf("No such page [%s]", page)
	}
	sections, err := eng.RenderSections(p.Story)
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
		fmt.Println("Added Action:", actionName)
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

func (eng *Engine) RenderActions() ([]*RenderedAction, error) {
	actions := eng.pages[eng.currentPage].Actions.Actions
	toret := []*RenderedAction{}
	for key, action := range actions {
		if action.Cond != nil {
			condMet, err := eng.runStrInEnvBool(eng.story, "engine", *action.Cond)
			if err != nil {
				return nil, err
			}
			if !condMet {
				continue
			}
		}
		rendered, err := eng.RenderString(action.Text)
		if err != nil {
			return nil, err
		}
		toret = append(toret, &RenderedAction{
			ActionStr:    key,
			RenderedText: rendered,
		})
	}
	sort.Slice(toret, func(i, j int) bool {
		return toret[i].ActionStr < toret[j].ActionStr
	})
	return toret, nil
}

func (eng *Engine) RunAction(actionStr string) error {
	action, ok := eng.pages[eng.currentPage].Actions.Actions[actionStr]
	if !ok {
		return fmt.Errorf("No such action [%s].", actionStr)
	}
	if action.Cond != nil {
		condMet, err := eng.runStrInEnvBool(eng.story, "engine", *action.Cond)
		if err != nil {
			return err
		}
		if !condMet {
			return fmt.Errorf("Condition not met [%s]", *action.Cond)
		}
	}
	if action.Execute != nil {
		err := eng.runStrInEnv(eng.story, "engine", *action.Execute)
		if err != nil {
			return err
		}
	}
	if action.TransitionTo != nil {
		if _, ok := eng.pages[*action.TransitionTo]; ok {
			eng.currentPage = *action.TransitionTo
		} else {
			return fmt.Errorf("No such page [%s]", *action.TransitionTo)
		}
	}
	return nil
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

func (eng *Engine) RenderString(rawStr string) (string, error) {
	matches := eng.exprRegex.FindAllStringSubmatch(rawStr, -1)
	toret := rawStr
	for _, match := range matches {
		str, err := eng.runStrInEnvStr(eng.story, "engine", match[1])
		if err != nil {
			return "", err
		}
		toret = strings.Replace(toret, match[0], str, 1)
	}
	return toret, nil
}

func (eng *Engine) RenderSections(story *Story) ([]string, error) {
	toret := []string{}
	for _, section := range story.Sections {
		next, err := eng.RenderString(section)
		if err != nil {
			return nil, err
		}
		toret = append(toret, next)
	}
	return toret, nil
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

type RenderedAction struct {
	ActionStr    string
	RenderedText string
}
