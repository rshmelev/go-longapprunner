package golongapprunner

import (
	"testing"
	"github.com/davecgh/go-spew/spew"
	"fmt"
)

func TestStringToArgs2(t *testing.T) {
	s := "hello world 111 run=\"1\\t2\" x'3'z a=\"yes"
	fmt.Println(s)
	r := StringToArgs2(s)
	spew.Dump(r)
}