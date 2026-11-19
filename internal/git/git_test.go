package git

import (
	"reflect"
	"testing"
)

func TestParseEntries(t *testing.T) {
	out := "global\x00file:/home/u/.gitconfig\x00aicm.model\nhaiku\x00" +
		"local\x00file:.git/config\x00aicm.model\nsonnet\x00" +
		"local\x00file:.git/config\x00aicm.prompt\nUse gitmoji.\nOne line.\x00" +
		"local\x00file:.git/config\x00aicm.ticketPattern\x00"
	want := []Entry{
		{"global", "/home/u/.gitconfig", "model", "haiku"},
		{"local", ".git/config", "model", "sonnet"},
		{"local", ".git/config", "prompt", "Use gitmoji.\nOne line."},
		{"local", ".git/config", "ticketpattern", ""},
	}
	if got := parseEntries(out); !reflect.DeepEqual(got, want) {
		t.Errorf("parseEntries =\n%q\nwant\n%q", got, want)
	}
	if got := parseEntries(""); len(got) != 0 {
		t.Errorf("empty output: %q", got)
	}
}
