package main

import (
	"log"
	"fmt"
	"bytes"
	"strings"
	"strconv"
	//"regexp"
	"reflect"
	"path/filepath"
	"io/ioutil"
	"text/template/parse"
)

var ctemplates []string
var tmpl_ptr_map map[string]interface{} = make(map[string]interface{})
var text_overlap_list map[string]int

func init() {
	text_overlap_list = make(map[string]int)
}

type VarItem struct
{
	Name string
	Destination string
	Type string
}

type VarItemReflect struct
{
	Name string
	Destination string
	Value reflect.Value
}

type CTemplateSet struct
{
	tlist map[string]*parse.Tree
	dir string
	funcMap map[string]interface{}
	importMap map[string]string
	Fragments map[string]int
	FragmentCursor map[string]int
	FragOut string
	varList map[string]VarItem
	localVars map[string]map[string]VarItemReflect
	stats map[string]int
	pVarList string
	pVarPosition int
	previousNode parse.NodeType
	currentNode parse.NodeType
	nextNode parse.NodeType
	//tempVars map[string]string
	doImports bool
	expectsInt interface{}
}

func (c *CTemplateSet) compile_template(name string, dir string, expects string, expectsInt interface{}, varList map[string]VarItem) (out string) {
	c.dir = dir
	c.doImports = true
	c.funcMap = map[string]interface{}{
		"and": "&&",
		"not": "!",
		"or": "||",
		"eq": true,
		"ge": true,
		"gt": true,
		"le": true,
		"lt": true,
		"ne": true,
		"add": true,
		"subtract": true,
		"multiply": true,
		"divide": true,
	}

	c.importMap = map[string]string{
		"io":"io",
		"strconv":"strconv",
	}
	c.varList = varList
	//c.pVarList = ""
	//c.pVarPosition = 0
	c.stats = make(map[string]int)
	c.expectsInt = expectsInt
	holdreflect := reflect.ValueOf(expectsInt)

	res, err := ioutil.ReadFile(dir + name)
	if err != nil {
		log.Fatal(err)
	}

	content := string(res)
	if minify_templates {
		content = minify(content)
	}

	tree := parse.New(name, c.funcMap)
	var treeSet map[string]*parse.Tree = make(map[string]*parse.Tree)
	tree, err = tree.Parse(content,"{{","}}", treeSet, c.funcMap)
	if err != nil {
		log.Fatal(err)
	}
	if super_debug {
		fmt.Println(name)
	}

	out = ""
	fname := strings.TrimSuffix(name, filepath.Ext(name))
	c.tlist = make(map[string]*parse.Tree)
	c.tlist[fname] = tree
	varholder := "tmpl_" + fname + "_vars"

	if super_debug {
		fmt.Println(c.tlist)
	}
	c.localVars = make(map[string]map[string]VarItemReflect)
	c.localVars[fname] = make(map[string]VarItemReflect)
	c.localVars[fname]["."] = VarItemReflect{".",varholder,holdreflect}
	if c.Fragments == nil {
		c.Fragments = make(map[string]int)
	}
	c.FragmentCursor = make(map[string]int)
	c.FragmentCursor[fname] = 0

	subtree := c.tlist[fname]
	if super_debug {
		fmt.Println(subtree.Root)
	}

	treeLength := len(subtree.Root.Nodes)
	for index, node := range subtree.Root.Nodes {
		if super_debug {
			fmt.Println("Node: " + node.String())
		}

		c.previousNode = c.currentNode
		c.currentNode = node.Type()
		if treeLength != (index + 1) {
			c.nextNode = subtree.Root.Nodes[index + 1].Type()
		}
		out += c.compile_switch(varholder, holdreflect, fname, node)
	}

	var importList string
	if c.doImports {
		for _, item := range c.importMap {
			importList += "import \"" + item + "\"\n"
		}
	}

	var varString string
	for _, varItem := range c.varList {
		varString += "var " + varItem.Name + " " + varItem.Type + " = " + varItem.Destination + "\n"
	}

	fout := "// Code generated by Gosora. More below:\n/* This file was automatically generated by the software. Please don't edit it as your changes may be overwritten at any moment. */\n"
	fout += "// +build !no_templategen\npackage main\n" + importList + c.pVarList + "\n"
	fout += "func init() {\n\ttemplate_" + fname +"_handle = template_" + fname + "\n\t//o_template_" + fname +"_handle = template_" + fname + "\n\tctemplates = append(ctemplates,\"" + fname + "\")\n\ttmpl_ptr_map[\"" + fname + "\"] = &template_" + fname + "_handle\n\ttmpl_ptr_map[\"o_" + fname + "\"] = template_" + fname + "\n}\n\n"
	fout += "func template_" + fname + "(tmpl_" + fname + "_vars " + expects + ", w io.Writer) {\n" + varString + out + "}\n"

	fout = strings.Replace(fout,`))
w.Write([]byte(`," + ",-1)
	fout = strings.Replace(fout,"` + `","",-1)
	//spstr := "`([:space:]*)`"
	//whitespace_writes := regexp.MustCompile(`(?s)w.Write\(\[\]byte\(`+spstr+`\)\)`)
	//fout = whitespace_writes.ReplaceAllString(fout,"")

	if debug {
		for index, count := range c.stats {
			fmt.Println(index + ": " + strconv.Itoa(count))
		}
		fmt.Println(" ")
	}

	if super_debug {
		fmt.Println("Output!")
		fmt.Println(fout)
	}
	return fout
}

func (c *CTemplateSet) compile_switch(varholder string, holdreflect reflect.Value, template_name string, node interface{}) (out string) {
	if super_debug {
		fmt.Println("in compile_switch")
	}
	switch node := node.(type) {
		case *parse.ActionNode:
			if super_debug {
				fmt.Println("Action Node")
			}
			if node.Pipe == nil {
				break
			}
			for _, cmd := range node.Pipe.Cmds {
				out += c.compile_subswitch(varholder, holdreflect, template_name, cmd)
			}
			return out
		case *parse.IfNode:
			if super_debug {
				fmt.Println("If Node:")
				fmt.Println("node.Pipe",node.Pipe)
			}

			var expr string
			for _, cmd := range node.Pipe.Cmds {
				if super_debug {
					fmt.Println("If Node Bit:",cmd)
					fmt.Println("If Node Bit Type:",reflect.ValueOf(cmd).Type().Name())
				}
				expr += c.compile_varswitch(varholder, holdreflect, template_name, cmd)
				if super_debug {
					fmt.Println("If Node Expression Step:",c.compile_varswitch(varholder, holdreflect, template_name, cmd))
				}
			}

			if super_debug {
				fmt.Println("If Node Expression:",expr)
			}

			c.previousNode = c.currentNode
			c.currentNode = parse.NodeList
			c.nextNode = -1
			if node.ElseList == nil {
				if super_debug {
					fmt.Println("Selected Branch 1")
				}
				return "if " + expr + " {\n" + c.compile_switch(varholder, holdreflect, template_name, node.List) + "}\n"
			} else {
				if super_debug {
					fmt.Println("Selected Branch 2")
				}
				return "if " + expr + " {\n" + c.compile_switch(varholder, holdreflect, template_name, node.List) + "} else {\n" + c.compile_switch(varholder, holdreflect, template_name, node.ElseList) + "}\n"
			}
		case *parse.ListNode:
			if super_debug {
				fmt.Println("List Node")
			}
			for _, subnode := range node.Nodes {
				out += c.compile_switch(varholder, holdreflect, template_name, subnode)
			}
			return out
		case *parse.RangeNode:
			if super_debug {
				fmt.Println("Range Node!")
				fmt.Println(node.Pipe)
			}

			var outVal reflect.Value
			for _, cmd := range node.Pipe.Cmds {
				if super_debug {
					fmt.Println("Range Bit:",cmd)
				}
				out, outVal = c.compile_reflectswitch(varholder, holdreflect, template_name, cmd)
			}

			if super_debug {
				fmt.Println("Returned:",out)
				fmt.Println("Range Kind Switch!")
			}

			switch outVal.Kind() {
				case reflect.Map:
					var item reflect.Value
					for _, key := range outVal.MapKeys() {
						item = outVal.MapIndex(key)
					}

					if node.ElseList != nil {
						out = "if len(" + out + ") != 0 {\nfor _, item := range " + out + " {\n" + c.compile_switch("item", item, template_name, node.List) + "}\n} else {\n" + c.compile_switch("item", item, template_name, node.ElseList) + "}\n"
					} else {
						out = "if len(" + out + ") != 0 {\nfor _, item := range " + out + " {\n" + c.compile_switch("item", item, template_name, node.List) + "}\n}"
					}
				case reflect.Slice:
					if outVal.Len() == 0 {
						panic("The sample data needs at-least one or more elements for the slices. We're looking into removing this requirement at some point!")
					}
					item := outVal.Index(0)
					out = "if len(" + out + ") != 0 {\nfor _, item := range " + out + " {\n" + c.compile_switch("item", item, template_name, node.List) + "}\n}"
				case reflect.Invalid:
					return ""
			}

			if node.ElseList != nil {
				out += " else {\n" + c.compile_switch(varholder, holdreflect, template_name, node.ElseList) + "}\n"
			} else {
				out += "\n"
			}
			return out
		case *parse.TemplateNode:
			return c.compile_subtemplate(varholder, holdreflect, node)
		case *parse.TextNode:
			c.previousNode = c.currentNode
			c.currentNode = node.Type()
			c.nextNode = 0
			tmpText := bytes.TrimSpace(node.Text)
			if len(tmpText) == 0 {
				return ""
			} else {
				//return "w.Write([]byte(`" + string(node.Text) + "`))\n"
				fragment_name := template_name + "_" + strconv.Itoa(c.FragmentCursor[template_name])
				_, ok := c.Fragments[fragment_name]
				if !ok {
					c.Fragments[fragment_name] = len(node.Text)
					c.FragOut += "var " + fragment_name + " []byte = []byte(`" + string(node.Text) + "`)\n"
				}
				c.FragmentCursor[template_name] = c.FragmentCursor[template_name] + 1
				return "w.Write(" + fragment_name + ")\n"
			}
		default:
			panic("Unknown Node in main switch")
	}
	return ""
}

func (c *CTemplateSet) compile_subswitch(varholder string, holdreflect reflect.Value, template_name string, node *parse.CommandNode) (out string) {
	if super_debug {
		fmt.Println("in compile_subswitch")
	}
	firstWord := node.Args[0]
	switch n := firstWord.(type) {
		case *parse.FieldNode:
			if super_debug {
				fmt.Println("Field Node:",n.Ident)
			}

			/* Use reflect to determine if the field is for a method, otherwise assume it's a variable. Variable declarations are coming soon! */
			cur := holdreflect

			var varbit string
			if cur.Kind() == reflect.Interface {
				cur = cur.Elem()
				varbit += ".(" + cur.Type().Name() + ")"
			}

			for _, id := range n.Ident {
				if super_debug {
					fmt.Println("Data Kind:",cur.Kind().String())
					fmt.Println("Field Bit:",id)
				}

				cur = cur.FieldByName(id)
				if cur.Kind() == reflect.Interface {
					cur = cur.Elem()
					/*if cur.Kind() == reflect.String && cur.Type().Name() != "string" {
						varbit = "string(" + varbit + "." + id + ")"*/
					//if cur.Kind() == reflect.String && cur.Type().Name() != "string" {
					if cur.Type().PkgPath() != "main" && cur.Type().PkgPath() != "" {
						c.importMap["html/template"] = "html/template"
						varbit += "." + id + ".(" + strings.TrimPrefix(cur.Type().PkgPath(),"html/") + "." + cur.Type().Name() + ")"
					} else {
						varbit += "." + id + ".(" + cur.Type().Name() + ")"
					}
				} else {
					varbit += "." + id
				}
				if super_debug {
					fmt.Println("End Cycle")
				}
			}
			out = c.compile_varsub(varholder + varbit, cur)

			for _, varItem := range c.varList {
				if strings.HasPrefix(out, varItem.Destination) {
					out = strings.Replace(out, varItem.Destination, varItem.Name, 1)
				}
			}
			return out
		case *parse.DotNode:
			if super_debug {
				fmt.Println("Dot Node:",node.String())
			}
			return c.compile_varsub(varholder, holdreflect)
		case *parse.NilNode:
			panic("Nil is not a command x.x")
		case *parse.VariableNode:
			if super_debug {
				fmt.Println("Variable Node:",n.String())
				fmt.Println(n.Ident)
			}
			varname, reflectVal := c.compile_if_varsub(n.String(), varholder, template_name, holdreflect)
			return c.compile_varsub(varname, reflectVal)
		case *parse.StringNode:
			return n.Quoted
		case *parse.IdentifierNode:
			if super_debug {
				fmt.Println("Identifier Node:",node)
				fmt.Println("Identifier Node Args:",node.Args)
			}
			return c.compile_varsub(c.compile_identswitch(varholder, holdreflect, template_name, node))
		default:
			fmt.Println("Unknown Kind:",reflect.ValueOf(firstWord).Elem().Kind())
			fmt.Println("Unknown Type:",reflect.ValueOf(firstWord).Elem().Type().Name())
			panic("I don't know what node this is")
	}
	return ""
}

func (c *CTemplateSet) compile_varswitch(varholder string, holdreflect reflect.Value, template_name string, node *parse.CommandNode) (out string) {
	if super_debug {
		fmt.Println("in compile_varswitch")
	}
	firstWord := node.Args[0]
	switch n := firstWord.(type) {
		case *parse.FieldNode:
			if super_debug {
				fmt.Println("Field Node:",n.Ident)
				for _, id := range n.Ident {
					fmt.Println("Field Bit:",id)
				}
			}

			/* Use reflect to determine if the field is for a method, otherwise assume it's a variable. Coming Soon. */
			return c.compile_boolsub(n.String(), varholder, template_name, holdreflect)
		case *parse.ChainNode:
			if super_debug {
				fmt.Println("Chain Node:",n.Node)
				fmt.Println("Chain Node Args:",node.Args)
			}
			break
		case *parse.IdentifierNode:
			if super_debug {
				fmt.Println("Identifier Node:",node)
				fmt.Println("Identifier Node Args:",node.Args)
			}
			return c.compile_identswitch_n(varholder, holdreflect, template_name, node)
		case *parse.DotNode:
			return varholder
		case *parse.VariableNode:
			if super_debug {
				fmt.Println("Variable Node:",n.String())
				fmt.Println("Variable Node Identifier:",n.Ident)
			}
			out, _ = c.compile_if_varsub(n.String(), varholder, template_name, holdreflect)
			return out
		case *parse.NilNode:
			panic("Nil is not a command x.x")
		case *parse.PipeNode:
			if super_debug {
				fmt.Println("Pipe Node!")
				fmt.Println(n)
				fmt.Println("Args:",node.Args)
			}
			out += c.compile_identswitch_n(varholder, holdreflect, template_name, node)

			if super_debug {
				fmt.Println("Out:",out)
			}
			return out
		default:
			fmt.Println("Unknown Kind:",reflect.ValueOf(firstWord).Elem().Kind())
			fmt.Println("Unknown Type:",reflect.ValueOf(firstWord).Elem().Type().Name())
			panic("I don't know what node this is! Grr...")
	}
	return ""
}

func (c *CTemplateSet) compile_identswitch_n(varholder string, holdreflect reflect.Value, template_name string, node *parse.CommandNode) (out string) {
	if super_debug {
		fmt.Println("in compile_identswitch_n")
	}
	out, _ = c.compile_identswitch(varholder, holdreflect, template_name, node)
	return out
}

func (c *CTemplateSet) compile_identswitch(varholder string, holdreflect reflect.Value, template_name string, node *parse.CommandNode) (out string, val reflect.Value) {
	if super_debug {
		fmt.Println("in compile_identswitch")
	}

	//var outbuf map[int]string
	ArgLoop:
	for pos := 0; pos < len(node.Args); pos++ {
		id := node.Args[pos]
		if super_debug {
			fmt.Println("pos:",pos)
			fmt.Println("ID:",id)
		}
		switch id.String() {
			case "not":
				out += "!"
			case "or":
				if super_debug {
					fmt.Println("Building or function")
				}
				if pos == 0 {
					fmt.Println("pos:",pos)
					panic("or is missing a left operand")
					return out, val
				}
				if len(node.Args) <= pos {
					fmt.Println("post pos:",pos)
					fmt.Println("len(node.Args):",len(node.Args))
					panic("or is missing a right operand")
					return out, val
				}

				left := c.compile_boolsub(node.Args[pos - 1].String(), varholder, template_name, holdreflect)
				_, funcExists := c.funcMap[node.Args[pos + 1].String()]

				var right string
				if !funcExists {
					right = c.compile_boolsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				}

				out += left + " || " + right

				if super_debug {
					fmt.Println("Left operand:",node.Args[pos - 1])
					fmt.Println("Right operand:",node.Args[pos + 1])
				}

				if !funcExists {
					pos++
				}

				if super_debug {
					fmt.Println("pos:",pos)
					fmt.Println("len(node.Args):",len(node.Args))
				}
			case "and":
				if super_debug {
					fmt.Println("Building and function")
				}
				if pos == 0 {
					fmt.Println("pos:",pos)
					panic("and is missing a left operand")
					return out, val
				}
				if len(node.Args) <= pos {
					fmt.Println("post pos:",pos)
					fmt.Println("len(node.Args):",len(node.Args))
					panic("and is missing a right operand")
					return out, val
				}

				left := c.compile_boolsub(node.Args[pos - 1].String(), varholder, template_name, holdreflect)
				_, funcExists := c.funcMap[node.Args[pos + 1].String()]

				var right string
				if !funcExists {
					right = c.compile_boolsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				}

				out += left + " && " + right

				if super_debug {
					fmt.Println("Left operand:",node.Args[pos - 1])
					fmt.Println("Right operand:",node.Args[pos + 1])
				}

				if !funcExists {
					pos++
				}

				if super_debug {
					fmt.Println("pos:",pos)
					fmt.Println("len(node.Args):",len(node.Args))
				}
			case "le":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " <= " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "lt":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " < " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "gt":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " > " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "ge":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " >= " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "eq":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " == " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "ne":
				out += c.compile_if_varsub_n(node.Args[pos + 1].String(), varholder, template_name, holdreflect) + " != " + c.compile_if_varsub_n(node.Args[pos + 2].String(), varholder, template_name, holdreflect)
				if super_debug {
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "add":
				param1, val2 := c.compile_if_varsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				param2, val3 := c.compile_if_varsub(node.Args[pos + 2].String(), varholder, template_name, holdreflect)

				if val2.IsValid() {
					val = val2
				} else if val3.IsValid() {
					val = val3
				} else {
					numSample := 1
					val = reflect.ValueOf(numSample)
				}

				out += param1 + " + " + param2
				if super_debug {
					fmt.Println("add")
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "subtract":
				param1, val2 := c.compile_if_varsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				param2, val3 := c.compile_if_varsub(node.Args[pos + 2].String(), varholder, template_name, holdreflect)

				if val2.IsValid() {
					val = val2
				} else if val3.IsValid() {
					val = val3
				} else {
					numSample := 1
					val = reflect.ValueOf(numSample)
				}

				out += param1 + " - " + param2
				if super_debug {
					fmt.Println("subtract")
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "divide":
				param1, val2 := c.compile_if_varsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				param2, val3 := c.compile_if_varsub(node.Args[pos + 2].String(), varholder, template_name, holdreflect)

				if val2.IsValid() {
					val = val2
				} else if val3.IsValid() {
					val = val3
				} else {
					numSample := 1
					val = reflect.ValueOf(numSample)
				}

				out += param1 + " / " + param2
				if super_debug {
					fmt.Println("divide")
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			case "multiply":
				param1, val2 := c.compile_if_varsub(node.Args[pos + 1].String(), varholder, template_name, holdreflect)
				param2, val3 := c.compile_if_varsub(node.Args[pos + 2].String(), varholder, template_name, holdreflect)

				if val2.IsValid() {
					val = val2
				} else if val3.IsValid() {
					val = val3
				} else {
					numSample := 1
					val = reflect.ValueOf(numSample)
				}

				out += param1 + " * " + param2
				if super_debug {
					fmt.Println("multiply")
					fmt.Println(node.Args[pos + 1])
					fmt.Println(node.Args[pos + 2])
				}
				break ArgLoop
			default:
				if super_debug {
					fmt.Println("Variable!")
				}
				if len(node.Args) > (pos + 1) {
					next_node := node.Args[pos + 1].String()
					if next_node == "or" || next_node == "and" {
						continue
					}
				}
				out += c.compile_if_varsub_n(id.String(), varholder, template_name, holdreflect)
		}
	}

	//for _, outval := range outbuf {
	//	out += outval
	//}
	return out, val
}

func (c *CTemplateSet) compile_reflectswitch(varholder string, holdreflect reflect.Value, template_name string, node *parse.CommandNode) (out string, outVal reflect.Value) {
	if super_debug {
		fmt.Println("in compile_reflectswitch")
	}
	firstWord := node.Args[0]
	switch n := firstWord.(type) {
		case *parse.FieldNode:
			if super_debug {
				fmt.Println("Field Node:",n.Ident)
				for _, id := range n.Ident {
					fmt.Println("Field Bit:",id)
				}
			}
			/* Use reflect to determine if the field is for a method, otherwise assume it's a variable. Coming Soon. */
			return c.compile_if_varsub(n.String(), varholder, template_name, holdreflect)
		case *parse.ChainNode:
			if super_debug {
				fmt.Println("Chain Node: ")
				fmt.Println(n.Node)
				fmt.Println(node.Args)
			}
			return "", outVal
		case *parse.DotNode:
			return varholder, holdreflect
		case *parse.NilNode:
			panic("Nil is not a command x.x")
		default:
			//panic("I don't know what node this is")
	}
	return "", outVal
}

func (c *CTemplateSet) compile_if_varsub_n(varname string, varholder string, template_name string, cur reflect.Value) (out string) {
	if super_debug {
		fmt.Println("in compile_if_varsub_n")
	}
	out, _ = c.compile_if_varsub(varname, varholder, template_name, cur)
	return out
}

func (c *CTemplateSet) compile_if_varsub(varname string, varholder string, template_name string, cur reflect.Value) (out string, val reflect.Value) {
	if super_debug {
		fmt.Println("in compile_if_varsub")
	}
	if varname[0] != '.' && varname[0] != '$' {
		return varname, cur
	}

	bits := strings.Split(varname,".")
	if varname[0] == '$' {
		var res VarItemReflect
		if varname[1] == '.' {
			res = c.localVars[template_name]["."]
		} else {
			res = c.localVars[template_name][strings.TrimPrefix(bits[0],"$")]
		}
		out += res.Destination
		cur = res.Value

		if cur.Kind() == reflect.Interface {
			cur = cur.Elem()
		}
	} else {
		if cur.Kind() == reflect.Interface {
			cur = cur.Elem()
			out += varholder + ".(" + cur.Type().Name() + ")"
		} else {
			out += varholder
		}
	}
	bits[0] = strings.TrimPrefix(bits[0],"$")

	if super_debug {
		fmt.Println("Cur Kind:",cur.Kind())
		fmt.Println("Cur Type:",cur.Type().Name())
	}

	for _, bit := range bits {
		if super_debug {
			fmt.Println("Variable Field!")
			fmt.Println(bit)
		}
		if bit == "" {
			continue
		}

		cur = cur.FieldByName(bit)
		if cur.Kind() == reflect.Interface {
			cur = cur.Elem()
			out += "." + bit + ".(" + cur.Type().Name() + ")"
		} else {
			out += "." + bit
		}

		if !cur.IsValid() {
			panic(out + "^\n" + "Invalid value. Maybe, it doesn't exist?")
		}

		if super_debug {
			fmt.Println("Data Kind:",cur.Kind())
			fmt.Println("Data Type:",cur.Type().Name())
		}
	}

	if super_debug {
		fmt.Println("Out Value:",out)
		fmt.Println("Out Kind:",cur.Kind())
		fmt.Println("Out Type:",cur.Type().Name())
	}

	for _, varItem := range c.varList {
		if strings.HasPrefix(out, varItem.Destination) {
			out = strings.Replace(out, varItem.Destination, varItem.Name, 1)
		}
	}

	if super_debug {
		fmt.Println("Out Value:",out)
		fmt.Println("Out Kind:",cur.Kind())
		fmt.Println("Out Type:",cur.Type().Name())
	}

	_, ok := c.stats[out]
	if ok {
		c.stats[out]++
	} else {
		c.stats[out] = 1
	}

	return out, cur
}

func (c *CTemplateSet) compile_boolsub(varname string, varholder string, template_name string, val reflect.Value) string {
	if super_debug {
		fmt.Println("in compile_boolsub")
	}
	out, val := c.compile_if_varsub(varname, varholder, template_name, val)
	switch val.Kind() {
		case reflect.Int: out += " > 0"
		case reflect.Bool: // Do nothing
		case reflect.String: out += " != \"\""
		case reflect.Int64: out += " > 0"
		default:
			fmt.Println("Variable Name:",varname)
			fmt.Println("Variable Holder:",varholder)
			fmt.Println("Variable Kind:",val.Kind())
			panic("I don't know what this variable's type is o.o\n")
	}
	return out
}

func (c *CTemplateSet) compile_varsub(varname string, val reflect.Value) string {
	if super_debug {
		fmt.Println("in compile_varsub")
	}
	for _, varItem := range c.varList {
		if strings.HasPrefix(varname, varItem.Destination) {
			varname = strings.Replace(varname, varItem.Destination, varItem.Name, 1)
		}
	}

	_, ok := c.stats[varname]
	if ok {
		c.stats[varname]++
	} else {
		c.stats[varname] = 1
	}

	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	switch val.Kind() {
		case reflect.Int:
			return "w.Write([]byte(strconv.Itoa(" + varname + ")))\n"
		case reflect.Bool:
			return "if " + varname + " {\nw.Write([]byte(\"true\"))} else {\nw.Write([]byte(\"false\"))\n}\n"
		case reflect.String:
			if val.Type().Name() != "string" && !strings.HasPrefix(varname,"string(") {
				return "w.Write([]byte(string(" + varname + ")))\n"
			} else {
				return "w.Write([]byte(" + varname + "))\n"
			}
		case reflect.Int64:
			return "w.Write([]byte(strconv.FormatInt(" + varname + ", 10)))"
		default:
			fmt.Println("Unknown Variable Name:",varname)
			fmt.Println("Unknown Kind:",val.Kind())
			fmt.Println("Unknown Type:",val.Type().Name())
			panic("// I don't know what this variable's type is o.o\n")
	}
}

func (c *CTemplateSet) compile_subtemplate(pvarholder string, pholdreflect reflect.Value, node *parse.TemplateNode) (out string) {
	if super_debug {
		fmt.Println("in compile_subtemplate")
		fmt.Println("Template Node: " + node.Name)
	}

	fname := strings.TrimSuffix(node.Name, filepath.Ext(node.Name))
	varholder := "tmpl_" + fname + "_vars"
	var holdreflect reflect.Value
	if node.Pipe != nil {
		for _, cmd := range node.Pipe.Cmds {
			firstWord := cmd.Args[0]
			switch firstWord.(type) {
				case *parse.DotNode:
					varholder = pvarholder
					holdreflect = pholdreflect
					break
				case *parse.NilNode:
					panic("Nil is not a command x.x")
				default:
					out = "var " + varholder + " := false\n"
					out += c.compile_command(cmd)
			}
		}
	}

	res, err := ioutil.ReadFile(c.dir + node.Name)
	if err != nil {
		log.Fatal(err)
	}

	content := string(res)
	if minify_templates {
		content = minify(content)
	}

	tree := parse.New(node.Name, c.funcMap)
	var treeSet map[string]*parse.Tree = make(map[string]*parse.Tree)
	tree, err = tree.Parse(content,"{{","}}", treeSet, c.funcMap)
	if err != nil {
		log.Fatal(err)
	}

	c.tlist[fname] = tree
	subtree := c.tlist[fname]
	if super_debug {
		fmt.Println(subtree.Root)
	}

	c.localVars[fname] = make(map[string]VarItemReflect)
	c.localVars[fname]["."] = VarItemReflect{".",varholder,holdreflect}
	c.FragmentCursor[fname] = 0

	treeLength := len(subtree.Root.Nodes)
	for index, node := range subtree.Root.Nodes {
		if super_debug {
			fmt.Println("Node:",node.String())
		}

		c.previousNode = c.currentNode
		c.currentNode = node.Type()
		if treeLength != (index + 1) {
			c.nextNode = subtree.Root.Nodes[index + 1].Type()
		}
		out += c.compile_switch(varholder, holdreflect, fname, node)
	}
	return out
}

func (c *CTemplateSet) compile_command(*parse.CommandNode) (out string) {
	panic("Uh oh! Something went wrong!")
	return ""
}

func minify(data string) string {
	data = strings.Replace(data,"\t","",-1)
	data = strings.Replace(data,"\v","",-1)
	data = strings.Replace(data,"\n","",-1)
	data = strings.Replace(data,"\r","",-1)
	data = strings.Replace(data,"  "," ",-1)
	return data
}
