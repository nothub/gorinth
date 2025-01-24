//usr/bin/env -S go run "$0" "$@" ; exit
//go:build exclude

package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"os/exec"
	"text/template"

	"github.com/nothub/mrpack-install/cmd"
)

var contribs = []Contrib{
	{
		Name: "Chikage0o0",
		URL:  "https://github.com/Chikage0o0",
	},
	{
		Name: "William Hergès",
		URL:  "https://github.com/anhgelus",
	},
	{
		Name: "Mohamed Tawous",
		URL:  "https://github.com/mmtawous",
	},
	{
		Name: "Pr. James Hunter",
		URL:  "https://github.com/Hunter200165",
	},
	{
		Name: "murder_spagurder",
		URL:  "https://github.com/murderspagurder",
	},
}

//go:embed readme.tmpl
var fs embed.FS

func init() {
	log.SetFlags(0)
}

type Data struct {
	Entries  []CmdEntry
	Contribs []Contrib
}

type CmdEntry struct {
	Name string
	Help string
}

type Contrib struct {
	Name string
	URL  string
}

func NewCmdEntry(name string, cmd string) CmdEntry {
	help, err := exec.Command("/tmp/mrpack-install", cmd, "--help").CombinedOutput()
	if err != nil {
		log.Fatalln(err.Error())
	}
	return CmdEntry{
		Name: name,
		Help: string(help),
	}
}

func main() {

	err := exec.Command("go", "build", "-o", "/tmp/mrpack-install").Run()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var data Data
	data.Contribs = contribs
	data.Entries = append(data.Entries, NewCmdEntry("root", ""))
	for _, sc := range cmd.RootCmd.Commands() {
		data.Entries = append(data.Entries, NewCmdEntry(sc.Name(), sc.Name()))
	}

	tmpl, err := template.ParseFS(fs, "readme.tmpl")
	if err != nil {
		log.Fatalln(err.Error())
	}

	var buf = bytes.Buffer{}
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Print(buf.String())

}
