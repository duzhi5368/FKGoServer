//---------------------------------------------
package framework

//---------------------------------------------

import (
	PROTO "FKGoServer/FKGRpc_WordFilter/Proto"
	BUFIO "bufio"
	OS "os"
	STRINGS "strings"
	UTF8 "unicode/utf8"

	CONTEXT "golang.org/x/net/context"

	LOG "github.com/Sirupsen/logrus"
	SEGO "github.com/huichen/sego"
)

//---------------------------------------------
const (
	SERVICE = "[WORDFILTER]"
)

//---------------------------------------------
var (
	replaceTo   = "*" //"▇" // "*"
	replaceByte = []byte(STRINGS.Repeat(replaceTo, 50))
)

//---------------------------------------------
type Server struct {
	dirty_words map[string]bool
	segmenter   SEGO.Segmenter
}

//---------------------------------------------
func (s *Server) Func_Init() {
	s.dirty_words = make(map[string]bool)

	dict_path, dirty_words_path := s.data_path()
	// 载入字典
	LOG.Debug("Loading Dictionary...")
	s.segmenter.LoadDictionary(dict_path)
	LOG.Debug("Dictionary Loaded")

	// 读取脏词库
	LOG.Debug("Loading Dirty Words...")
	f, err := OS.Open(dirty_words_path)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	defer f.Close()

	// 逐行扫描
	scanner := BUFIO.NewScanner(f)
	scanner.Split(BUFIO.ScanLines)
	for scanner.Scan() {
		words := STRINGS.Split(STRINGS.ToUpper(STRINGS.TrimSpace(scanner.Text())), " ") // 均处理为大写
		if words[0] != "" {
			s.dirty_words[words[0]] = true
		}
	}
	LOG.Debug("Dirty Words Loaded")
}

//---------------------------------------------
// get correct dict path from GOPATH
func (s *Server) data_path() (dict_path string, dirty_words_path string) {
	paths := STRINGS.Split(OS.Getenv("GOPATH"), ";")
	for k := range paths {
		dirty_words_path = paths[k] + "/src/FKGoServer/FKGRpc_WordFilter/Res/dirty.txt"
		_, err := OS.Lstat(dirty_words_path)
		if err == nil {
			dict_path = paths[k] + "/src/FKGoServer/FKGRpc_WordFilter/Res/dirty.txt," + paths[k] + "/src/FKGoServer/FKGRpc_WordFilter/Res/dictionary.txt"
			return
		}
	}
	return
}

//---------------------------------------------
func (s *Server) Filter(ctx CONTEXT.Context, in *PROTO.WordFilter_Text) (*PROTO.WordFilter_Text, error) {
	bin := []byte(in.Text)
	segments := s.segmenter.Segment(bin)
	clean_text := make([]byte, 0, len(bin))
	for _, seg := range segments {
		word := bin[seg.Start():seg.End()]
		if s.dirty_words[STRINGS.ToUpper(string(word))] {
			clean_text = append(clean_text, replaceByte[:UTF8.RuneCount(word)]...)
		} else {
			clean_text = append(clean_text, word...)
		}
	}
	return &PROTO.WordFilter_Text{string(clean_text)}, nil
}

//---------------------------------------------
