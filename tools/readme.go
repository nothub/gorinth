//usr/bin/env -S go run "$0" "$@" ; exit
//go:build exclude

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/nothub/mrpack-install/cmd"
	"log"
	"net/http"
	"os/exec"
	"text/template"
)

//go:embed readme.tmpl
var fs embed.FS

func init() {
	log.SetFlags(0)
}

func main() {

	var data struct {
		Entries  []command
		Contribs []ghUser
	}

	// extract command infos
	err := exec.Command("go", "build", "-o", "/tmp/mrpack-install").Run()
	if err != nil {
		log.Fatalln(err.Error())
	}
	data.Entries = append(data.Entries, extractCmdInfos("root", ""))
	for _, sc := range cmd.RootCmd.Commands() {
		data.Entries = append(data.Entries, extractCmdInfos(sc.Name(), sc.Name()))
	}

	// fetch contribs from gh api
	data.Contribs, err = getContribs("nothub", "mrpack-install")
	if err != nil {
		log.Fatalln(err.Error())
	}
	data.Contribs = filterBots(data.Contribs)

	// load template
	tmpl, err := template.ParseFS(fs, "readme.tmpl")
	if err != nil {
		log.Fatalln(err.Error())
	}

	// apply data to template
	var buf = bytes.Buffer{}
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Print(buf.String())
}

type command struct {
	Name string
	Help string
}

func extractCmdInfos(name string, cmd string) command {
	help, err := exec.Command("/tmp/mrpack-install", cmd, "--help").CombinedOutput()
	if err != nil {
		log.Fatalln(err.Error())
	}
	return command{
		Name: name,
		Help: string(help),
	}
}

type ghUser struct {
	Login         string `json:"login"`
	Name          string `json:"name"`
	HtmlUrl       string `json:"html_url"`
	AvatarUrl     string `json:"avatar_url"`
	Contributions int    `json:"contributions"`
}

func getContribs(owner string, repo string) (contribs []ghUser, err error) {

	err = ghApi(fmt.Sprintf("/repos/%s/%s/contributors", owner, repo), &contribs)
	if err != nil {
		return nil, err
	}

	// 'contributors' endpoint is not enough, it does not provide names...
	// we have to fetch them from the 'users' endpoint:
	for i, contrib := range contribs {

		err = ghApi(fmt.Sprintf("/users/%s", contrib.Login), &contrib)
		if err != nil {
			return nil, err
		}

		if contrib.Name == "" {
			contrib.Name = contrib.Login
		}

		contribs[i] = contrib
	}

	return contribs, nil
}

func ghApi(endpoint string, target interface{}) error {

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com"+endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(target)
}

func filterBots(list []ghUser) []ghUser {
	result := make([]ghUser, 0, len(list))

	for _, usr := range list {
		if usr.Login != "dependabot[bot]" {
			result = append(result, usr)
		}
	}

	return result
}
