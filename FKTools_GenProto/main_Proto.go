//---------------------------------------------
package main

//---------------------------------------------
import (
	BUFIO "bufio"
	BYTES "bytes"
	JSON "encoding/json"
	IO "io"
	IOUTIL "io/ioutil"
	LOG "log"
	OS "os"
	REGEXP "regexp"
	TEMPLATE "text/template"
	UNICODE "unicode"

	CLI "gopkg.in/urfave/cli.v2"
)

//---------------------------------------------
const (
	TK_SYMBOL = iota
	TK_STRUCT_BEGIN
	TK_STRUCT_END
	TK_DATA_TYPE
	TK_ARRAY
	TK_EOF
)

//---------------------------------------------
var (
	datatypes map[string]map[string]struct {
		T string `json:"t"` // type
		R string `json:"r"` // read
		W string `json:"w"` // write
	} // type -> language -> t/r/w
)

//---------------------------------------------
var (
	TOKEN_EOF = &token{typ: TK_EOF}
)

//---------------------------------------------
type (
	field_info struct {
		Name  string
		Typ   string
		Array bool
	}
	struct_info struct {
		Name   string
		Fields []field_info
	}
)

//---------------------------------------------
type token struct {
	typ     int
	literal string
	r       rune
}

//---------------------------------------------
func syntax_error(p *Parser) {
	LOG.Println("syntax error @line:", p.lexer.lineno)
	LOG.Println(">> \033[1;31m", p.lexer.lines[p.lexer.lineno-1], "\033[0m <<")
	OS.Exit(-1)
}

//---------------------------------------------
type Lexer struct {
	reader *BYTES.Buffer
	lines  []string
	lineno int
}

//---------------------------------------------
func (lex *Lexer) init(r IO.Reader) {
	bts, err := IOUTIL.ReadAll(r)
	if err != nil {
		LOG.Fatal(err)
	}

	// 按行读入源码
	scanner := BUFIO.NewScanner(BYTES.NewBuffer(bts))
	for scanner.Scan() {
		lex.lines = append(lex.lines, scanner.Text())
	}

	// 清除注释
	re := REGEXP.MustCompile("(?m:^#(.*)$)")
	bts = re.ReplaceAllLiteral(bts, nil)
	lex.reader = BYTES.NewBuffer(bts)
	lex.lineno = 1
}

//---------------------------------------------
func (lex *Lexer) next() (t *token) {
	defer func() {
		//log.Println(t)
	}()
	var r rune
	var err error
	for {
		r, _, err = lex.reader.ReadRune()
		if err == IO.EOF {
			return TOKEN_EOF
		} else if UNICODE.IsSpace(r) {
			if r == '\n' {
				lex.lineno++
			}
			continue
		}
		break
	}

	if r == '=' {
		for k := 0; k < 2; k++ { // check "==="
			r, _, err = lex.reader.ReadRune()
			if err == IO.EOF {
				return TOKEN_EOF
			}
			if r != '=' {
				lex.reader.UnreadRune()
				return &token{typ: TK_STRUCT_BEGIN}
			}
		}
		return &token{typ: TK_STRUCT_END}
	} else if UNICODE.IsLetter(r) {
		var runes []rune
		for {
			runes = append(runes, r)
			r, _, err = lex.reader.ReadRune()
			if err == IO.EOF {
				break
			} else if UNICODE.IsLetter(r) || UNICODE.IsNumber(r) || r == '_' {
				continue
			} else {
				lex.reader.UnreadRune()
				break
			}
		}

		t := &token{}
		t.literal = string(runes)
		if _, ok := datatypes[t.literal]; ok {
			t.typ = TK_DATA_TYPE
		} else if t.literal == "array" {
			t.typ = TK_ARRAY
		} else {
			t.typ = TK_SYMBOL
		}

		return t
	} else {
		LOG.Fatal("lex error @line:", lex.lineno)
	}
	return nil
}

//---------------------------------------------
func (lex *Lexer) eof() bool {
	for {
		r, _, err := lex.reader.ReadRune()
		if err == IO.EOF {
			return true
		} else if UNICODE.IsSpace(r) {
			if r == '\n' {
				lex.lineno++
			}
			continue
		} else {
			lex.reader.UnreadRune()
			return false
		}
	}
}

//////////////////////////////////////////////////////////////
type Parser struct {
	lexer   *Lexer
	infos   []struct_info
	symbols map[string]bool
}

//---------------------------------------------
func (p *Parser) init(lex *Lexer) {
	p.lexer = lex
	p.symbols = make(map[string]bool)
}

//---------------------------------------------
func (p *Parser) match(typ int) *token {
	t := p.lexer.next()
	if t.typ != typ {
		syntax_error(p)
	}
	return t
}

//---------------------------------------------
func (p *Parser) expr() bool {
	if p.lexer.eof() {
		return false
	}
	info := struct_info{}

	t := p.match(TK_SYMBOL)
	info.Name = t.literal
	p.symbols[t.literal] = true
	p.match(TK_STRUCT_BEGIN)
	p.fields(&info)
	p.infos = append(p.infos, info)
	return true
}

//---------------------------------------------
func (p *Parser) fields(info *struct_info) {
	for {
		t := p.lexer.next()
		if t.typ == TK_STRUCT_END {
			return
		}
		if t.typ != TK_SYMBOL {
			syntax_error(p)
		}

		field := field_info{Name: t.literal}
		t = p.lexer.next()
		if t.typ == TK_ARRAY {
			field.Array = true
			t = p.lexer.next()
		}

		if t.typ == TK_DATA_TYPE || t.typ == TK_SYMBOL {
			field.Typ = t.literal
		} else {
			syntax_error(p)
		}

		info.Fields = append(info.Fields, field)
	}
}

//---------------------------------------------
func (p *Parser) semantic_check() {
	for _, info := range p.infos {
	FIELDLOOP:
		for _, field := range info.Fields {
			if _, ok := datatypes[field.Typ]; !ok {
				if p.symbols[field.Typ] {
					continue FIELDLOOP
				}
				LOG.Fatal("symbol not found:", field)
			}
		}
	}
}

//---------------------------------------------
func main() {
	app := &CLI.App{
		Name:  "Protocol Data Structure Generator",
		Usage: "handle proto.txt",
		Authors: []CLI.Author{
			{Name: "xtaci"},
			{Name: "ycs"},
		},
		Version: "1.0",
		Flags: []CLI.Flag{
			&CLI.StringFlag{Name: "file,f", Value: "./proto.txt", Usage: "input proto.txt file"},
			&CLI.StringFlag{Name: "binding,b", Value: "go", Usage: `language type binding:"go","cs"`},
			&CLI.StringFlag{Name: "template,t", Value: "./templates/server/proto.tmpl", Usage: "template file"},
			&CLI.StringFlag{Name: "pkgname", Value: "agent", Usage: "package name to prefix"},
		},
		Action: func(c *CLI.Context) error {
			// load primitives mapping
			f, err := OS.Open("primitives.json")
			if err != nil {
				LOG.Fatal(err)
			}
			if err := JSON.NewDecoder(f).Decode(&datatypes); err != nil {
				LOG.Fatal(err)
			}

			// parse
			file, err := OS.Open(c.String("file"))
			if err != nil {
				LOG.Fatal(err)
			}
			lexer := Lexer{}
			lexer.init(file)
			p := Parser{}
			p.init(&lexer)
			for p.expr() {
			}

			// semantic
			p.semantic_check()

			// use template to generate final output
			funcMap := TEMPLATE.FuncMap{
				"Type": func(t string) string {
					return datatypes[t][c.String("binding")].T
				},
				"Read": func(t string) string {
					return datatypes[t][c.String("binding")].R
				},
				"Write": func(t string) string {
					return datatypes[t][c.String("binding")].W
				},
			}
			tmpl, err := TEMPLATE.New("proto.tmpl").Funcs(funcMap).ParseFiles(c.String("template"))
			if err != nil {
				LOG.Fatal(err)
			}
			args := struct {
				PackageName string
				Infos       []struct_info
			}{c.String("pkgname"), p.infos}

			err = tmpl.Execute(OS.Stdout, args)
			if err != nil {
				LOG.Fatal(err)
			}
			return nil
		},
	}
	app.Run(OS.Args)
}
