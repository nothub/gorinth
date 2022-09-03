package cmd

import (
	"github.com/nothub/mrpack-install/http"
	modrinth "github.com/nothub/mrpack-install/modrinth/api"
	"github.com/nothub/mrpack-install/modrinth/mrpack"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	// TODO: global flags
	// rootCmd.PersistentFlags().BoolP("version", "V", false, "Print version infos")
	// rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("host", "api.modrinth.com", "Labrinth host")

	rootCmd.Flags().String("dir", "server", "Server directory path")
}

var rootCmd = &cobra.Command{
	Use:   "mrpack-install (<filepath>|<url>|<project-id> [<version>]|<project-slug> [<version>])",
	Short: "Modrinth Modpack server deployment",
	Long: `A cli application for installing Minecraft servers and Modrinth modpacks.
Requires a mrpack file path, a modrinth url or project id as argument.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		host, err := cmd.PersistentFlags().GetString("host")
		if err != nil {
			log.Fatalln(err)
		}

		input := args[0]
		version := ""
		if len(args) > 1 {
			version = args[1]
		}

		filePath := ""
		if isFilePath(input) {
			filePath = input

		} else if isUrl(input) {
			log.Println("Downloading mrpack file from", args)
			file, err := http.Instance.DownloadFile(input, ".")
			if err != nil {
				log.Fatalln(err)
			}
			filePath = file

		} else { // input is project id or slug?
			versions, err := modrinth.NewClient(host).GetProjectVersions(input, nil)
			if err != nil {
				log.Fatalln(err)
			}

			// get files uploaded for specified version or latest stable if not specified
			var files []modrinth.File
			for i := range versions {
				if version != "" {
					if versions[i].VersionNumber == version {
						files = versions[i].Files
						break
					}
				} else {
					if versions[i].VersionType == modrinth.ReleaseVersionType {
						files = versions[i].Files
						break
					}
				}
			}
			if len(files) == 0 {
				log.Fatalln("No files found for", input, version)
			}

			for i := range files {
				if strings.HasSuffix(files[i].Filename, ".mrpack") {
					log.Println("Downloading mrpack file from", files[i].Url)
					file, err := http.Instance.DownloadFile(files[i].Url, ".")
					if err != nil {
						log.Fatalln(err)
					}
					filePath = file
					break
				}
			}
			if filePath == "" {
				log.Fatalln("No mrpack file found for", input, version)
			}
		}

		if filePath == "" {
			log.Fatalln("An error occured!")
		}

		log.Println("Processing mrpack file", filePath)

		index, err := mrpack.ReadIndex(filePath)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Installing", index.Name)

		log.Printf("loader dependencies: %+v\n", index.Dependencies)
		// TODO: download server/loader dependencies

		log.Println("file dependencies:", len(index.Files))
		// TODO: download mods

		err = mrpack.ExtractOverrides(filePath, "testing")
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func isFilePath(s string) bool {
	_, err := os.Stat(s)
	return err == nil
}

func isUrl(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme == "" {
		return false
	}
	return true
}

func Execute() {
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}
