package update

import (
	"archive/zip"
	"fmt"
	"github.com/nothub/mrpack-install/requester"
	"github.com/nothub/mrpack-install/update/model"
	"github.com/nothub/mrpack-install/util"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DetectList map[model.Path]util.DetectType

// PreDelete Three scenarios
// 1.File does not exist Notice
// 2.File exists but hash value does not match,Change the original file name to xxx.bak
// 3.File exists and the hash value matches
func PreDelete(deleteList *model.ModPackInfo) DetectList {
	detectType := make(DetectList, 10)
	for filePath := range deleteList.File {
		switch util.FileDetection(deleteList.File[filePath].Hash, string(filePath)) {
		case util.PathMatchHashMatch:
			fmt.Printf("%s will remove\n", filePath)
			detectType[filePath] = util.PathMatchHashMatch
		case util.PathMatchHashNoMatch:
			fmt.Printf("%s will remove,The original file will be move to updateBack folder", filePath)
			detectType[filePath] = util.PathMatchHashNoMatch
		case util.PathNoMatch:
			fmt.Printf("%s isn't exist\n", filePath)
		}
	}
	return detectType
}

// PreUpdate Three scenarios
// 1.File does not exist
// 2.File exists but hash value does not match,Change the original file name to xxx.bak
// 3.File exists and the hash value matches,Remove the item from the queue
func PreUpdate(updateList *model.ModPackInfo) DetectList {
	detectType := make(DetectList, 10)
	for filePath := range updateList.File {
		switch util.FileDetection(updateList.File[filePath].Hash, string(filePath)) {
		case util.PathMatchHashMatch:
			delete(updateList.File, filePath)
		case util.PathMatchHashNoMatch:
			fmt.Printf("%s will update,The original file will be move to updateBack folder", filePath)
			detectType[filePath] = util.PathMatchHashNoMatch
		case util.PathNoMatch:
			fmt.Printf("%s will download\n", filePath)
			detectType[filePath] = util.PathNoMatch
		}
	}
	return detectType
}

func ModPackDeleteDo(deleteList DetectList, serverPath string) {
	for filePath := range deleteList {
		switch deleteList[filePath] {
		case util.PathMatchHashMatch:
			err := os.Remove(filepath.Join(serverPath, string(filePath)))
			if err != nil {
				fmt.Println(err)
			}
		case util.PathMatchHashNoMatch:
			err := os.Rename(filepath.Join(serverPath, string(filePath)), filepath.Join(serverPath, "updateBack", string(filePath)))
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func ModPackUpdateDo(updateList DetectList, updateFileInfo model.FileMap, serverPath string, modPackPath string, downloadPools requester.DownloadPools) error {
	//backup file and download file in modrinth index
	for filePath := range updateList {
		switch updateList[filePath] {
		case util.PathNoMatch:
			if updateFileInfo[filePath].DownloadLink != nil {
				downloadPools.Downloads = append(downloadPools.Downloads, requester.NewDownload(updateFileInfo[filePath].DownloadLink, map[string]string{"sha1": updateFileInfo[filePath].Hash}, filepath.Base(string(filePath)), filepath.Join(serverPath, filepath.Dir(string(filePath)))))
			}
		case util.PathMatchHashNoMatch:
			err := os.Rename(filepath.Join(serverPath, string(filePath)), filepath.Join(serverPath, "updateBack", string(filePath)))
			if err != nil {
				return err
			}
			if updateFileInfo[filePath].DownloadLink != nil {
				downloadPools.Downloads = append(downloadPools.Downloads, requester.NewDownload(updateFileInfo[filePath].DownloadLink, map[string]string{"sha1": updateFileInfo[filePath].Hash}, filepath.Base(string(filePath)), filepath.Join(serverPath, filepath.Dir(string(filePath)))))
			}
		}
	}
	downloadPools.Do()

	// unzip update file
	r, err := zip.OpenReader(modPackPath)
	if err != nil {
		return err
	}
	defer func(r *zip.ReadCloser) {
		err := r.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(r)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		filePathInZip := f.Name
		if strings.HasPrefix(filePathInZip, "overrides/") {
			filePathInZip = strings.TrimPrefix(filePathInZip, "overrides/")
		} else if strings.HasPrefix(filePathInZip, "server-overrides/") {
			filePathInZip = strings.TrimPrefix(filePathInZip, "server-overrides/")
		} else {
			continue
		}

		if _, ok := updateFileInfo[model.Path(filePathInZip)]; ok && updateFileInfo[model.Path(filePathInZip)].DownloadLink == nil {

			targetPath := filepath.Join(serverPath, filePathInZip)

			err := os.MkdirAll(filepath.Dir(targetPath), 0755)
			if err != nil {
				return err
			}

			fileReader, err := f.Open()
			if err != nil {
				return err
			}

			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, fileReader); err != nil {
				return err
			}

			err = fileReader.Close()
			if err != nil {
				return err
			}
			err = outFile.Close()
			if err != nil {
				return err
			}

			fmt.Println("Override file extracted:", targetPath)

		}
	}
	return nil
}
