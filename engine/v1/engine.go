package engine

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aarzilli/golua/lua"
	"github.com/alecthomas/participle"
	"github.com/stevedonovan/luar"
)

type StoryIn string
type StoryOut string
type Action string

type Scalar struct {
	Name string `"$"@Ident`
}

type ScalarSet struct {
	Key string `@Ident "="`
	Val Expr   ` @@ `
}

type Value struct {
	Str    *string  `@String | @RawString`
	Num    *float64 `| (@Float | @Int)`
	Scalar *Scalar  `| @@`
}

type Func struct {
	FuncName string  `@Ident "("`
	Params   []*Expr `[@@{"," @@}]")"`
}

type Expr struct {
	Value     *Value     `@@`
	Func      *Func      `| @@`
	ScalarSet *ScalarSet `| @@`
}

type ExprList struct {
	ExprList []*Expr `@@{";" @@}`
}

type Engine struct {
	luaDir       string
	storyDir     string
	storyPage    string
	exprBegin    string
	exprEnd      string
	exprRegex    *regexp.Regexp
	storyParser  *participle.Parser
	actionParser *participle.Parser
	state        *lua.State
	actionChan   chan Action
	storyChan    chan StoryOut
}

func NewEngine(storyDir, luaDir string) (*Engine, error) {
	toret := &Engine{
		luaDir:     luaDir,
		storyDir:   storyDir,
		storyPage:  "init",
		exprBegin:  `<[`,
		exprEnd:    `]>`,
		state:      luar.Init(),
		actionChan: make(chan Action, 8),
		storyChan:  make(chan StoryOut, 8),
	}
	err := toret.build()
	if err != nil {
		return nil, err
	}
	return toret, nil
}

func (adv *Engine) evalCurrentPage() (StoryOut, error) {
	file, err := ioutil.ReadFile(filepath.Join(adv.storyDir, adv.storyPage+".page"))
	if err != nil {
		return "", err
	}
	return adv.evalStory(StoryIn(file))
}

func (adv *Engine) sendCurrentStory() error {
	story, err := adv.evalCurrentPage()
	if err != nil {
		return err
	}
	adv.storyChan <- story
	return nil
}

func (adv *Engine) Run() (chan<- Action, <-chan StoryOut) {
	go func() {
		for action := range adv.actionChan {
			err := adv.evalAction(action)
			if err != nil {
				fmt.Printf("ERR: Not a valid action [%s]\n\t%s\n", action, err.Error())
			}
			err = adv.sendCurrentStory()
			if err != nil {
				fmt.Printf("ERR: Couldn't render story page [%s]\n\t%s\n", adv.storyPage, err.Error())
			}
		}
	}()
	err := adv.sendCurrentStory()
	if err != nil {
		fmt.Printf("ERR: Couldn't render story page [%s]\n\t%s\n", adv.storyPage, err.Error())
	}
	return adv.actionChan, adv.storyChan
}

func (adv *Engine) loadLua() error {
	err := filepath.Walk(adv.luaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".lua") {
			err = adv.state.DoFile(path)
			if err != nil {
				return err
			}
			fmt.Printf("Loaded Lua File: [%s]\n", info.Name())
			return nil
		}
		return nil
	})
	if err != nil {
		return err
	}
	luar.Register(adv.state, "engine", luar.Map{
		"SSP": adv.setStoryPage,
	})
	return nil
}

func (adv *Engine) loadParsers() error {
	prefix := regexp.QuoteMeta(adv.exprBegin)
	suffix := regexp.QuoteMeta(adv.exprEnd)
	r, err := regexp.Compile(prefix + `(?s)(.+?)` + suffix)
	if err != nil {
		return err
	}
	adv.exprRegex = r

	parser, err := participle.Build(&ExprList{})
	if err != nil {
		return err
	}
	adv.storyParser = parser

	parser, err = participle.Build(&Func{})
	if err != nil {
		return err
	}

	adv.actionParser = parser
	return nil
}

func (adv *Engine) loadStory() error {
	return nil
}

func (adv *Engine) build() error {
	err := adv.loadParsers()
	if err != nil {
		return err
	}
	err = adv.loadStory()
	if err != nil {
		return err
	}
	return adv.loadLua()
}

func (adv *Engine) setStoryPage(storyPage string) {
	adv.storyPage = storyPage
}

func (adv *Engine) evalStory(str StoryIn) (StoryOut, error) {
	toret := string(str)
	matches := adv.exprRegex.FindAllStringSubmatch(toret, -1)
	for _, match := range matches {
		result, err := adv.parseExpr(match[1])
		if err != nil {
			return "", err
		}
		last := ""
		for _, expr := range result.ExprList {
			last, err = adv.evaluateExprToStr(expr)
			if err != nil {
				return "", err
			}
		}
		toret = strings.Replace(toret, match[0], last, 1)
	}
	return StoryOut(toret), nil
}

func (adv *Engine) evalAction(act Action) error {
	action := &Func{}
	err := adv.actionParser.ParseString(string(act), action)
	if err != nil {
		return err
	}
	obj, err := adv.evaluateFuncToLua(action)
	if err != nil {
		return err
	}
	defer obj.Close()
	return nil
}

func (adv *Engine) parseExpr(expr string) (*ExprList, error) {
	f := &ExprList{}
	err := adv.storyParser.ParseString(expr, f)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (adv *Engine) luaObjToString(obj *luar.LuaObject) (string, error) {
	toret := []string{}
	tostr := luar.NewLuaObjectFromName(adv.state, "tostring")
	defer tostr.Close()
	err := tostr.Call(&toret, obj)
	if err != nil {
		return "", err
	}
	return toret[0], nil
}

func (adv *Engine) setScalar(key string, obj *luar.LuaObject) error {
	scalars := luar.NewLuaObjectFromName(adv.state, "scalars")
	defer scalars.Close()
	return scalars.Set(obj, key)
}

func (adv *Engine) evaluateExprToStr(expr *Expr) (string, error) {
	obj, err := adv.evaluateExprToLua(expr)
	if err != nil {
		return "", err
	}
	defer obj.Close()
	return adv.luaObjToString(obj)
}

func (adv *Engine) evaluateValueToLua(value *Value) (*luar.LuaObject, error) {
	switch {
	case value.Scalar != nil:
		return luar.NewLuaObjectFromName(adv.state, "scalars", value.Scalar.Name), nil
	case value.Num != nil:
		return luar.NewLuaObjectFromValue(adv.state, value.Num), nil
	case value.Str != nil:
		return luar.NewLuaObjectFromValue(adv.state, value.Str), nil
	}
	return nil, fmt.Errorf("Not a valid Value: %+v", *value)
}

func (adv *Engine) evaluateFuncToLua(f *Func) (*luar.LuaObject, error) {
	params := []interface{}{}
	fobj := luar.NewLuaObjectFromName(adv.state, f.FuncName)
	defer fobj.Close()
	for _, param := range f.Params {
		val, err := adv.evaluateExprToLua(param)
		if err != nil {
			return nil, err
		}
		defer val.Close()
		params = append(params, val)
	}
	toret := []interface{}{}
	err := fobj.Call(&toret, params...)
	if err != nil {
		return nil, err
	}
	obj := luar.NewLuaObjectFromValue(adv.state, toret[0])
	return obj, nil
}

func (adv *Engine) evaluateScalarSetToLua(sset *ScalarSet) (*luar.LuaObject, error) {
	obj, err := adv.evaluateExprToLua(&sset.Val)
	if err != nil {
		return nil, err
	}
	err = adv.setScalar(sset.Key, obj)
	if err != nil {
		obj.Close()
		return nil, err
	}
	return obj, nil
}

func (adv *Engine) evaluateExprToLua(expr *Expr) (*luar.LuaObject, error) {
	switch {
	case expr.Value != nil:
		return adv.evaluateValueToLua(expr.Value)
	case expr.Func != nil:
		return adv.evaluateFuncToLua(expr.Func)
	case expr.ScalarSet != nil:
		return adv.evaluateScalarSetToLua(expr.ScalarSet)
	}
	return nil, fmt.Errorf("Not yet implemented: %+v", *expr)
}
