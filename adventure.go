package main

import (
	"adventure/engine"
	"fmt"
)

const story = `
<[
get_scalar("inventory", new_table());
""]>
Or perhaps a <[ $test ]>
Somethign about a table <[ get_scalar("table", new_table()); table_index("table", "name", "Talon") ]>
Once upon a time <[ "testing" ]> there was <[ test = "dinosaur" ]>
And then there was a <[ sum( 1.2, 2.2 ) ]>
Or perhaps dancing <[ if_else( $test, $test, "t-rex" ) ]>
Or perhaps dancing <[ get_scalar("testk", "velociraptor") ]>
Or perhaps a <[ $testk; $test ]> <[ testi = sum( 1.2, 2.2 ) ]>
And <[printR( ` + "`" + `hahaha
ahahaha fasfdlfkj asdfsdf
asdfasdf` + "`" + ` )]>`

func main() {
	fmt.Println(story)
	fmt.Println("-------")
	adv, err := engine.NewEngine("./stories/test", "./lua")
	if err != nil {
		panic(err)
	}
	final, err := adv.EvalStory(story)
	if err != nil {
		panic(err)
	}
	fmt.Println(final)
	//expr := &Expr{}
	//parser, err := participle.Build(expr)
	//if err != nil {
	//	panic(err)
	//}
	//err = parser.ParseString(`$(Test)`, expr)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("%+v\n", expr)
}
