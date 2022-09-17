package update

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/nothub/mrpack-install/modrinth/mrpack"
	"github.com/nothub/mrpack-install/update/model"
	"github.com/nothub/mrpack-install/util"
	"reflect"
	"strings"
)

func GenerateModPackInfo(modPackPatch string) (*model.ModPackInfo, error) {
	var modPackInfo model.ModPackInfo
	modPackInfo.File = make(model.FileMap)

	modrinthIndex, err := mrpack.ReadIndex(modPackPatch)
	if err != nil {
		return nil, err
	}

	modPackInfo.Dependencies = modrinthIndex.Dependencies
	modPackInfo.ModPackVersion = modrinthIndex.VersionId
	modPackInfo.ModPackName = modrinthIndex.Name

	// Add modrinth.index file
	for _, file := range modrinthIndex.Files {
		modPackInfo.File[model.Path(file.Path)] = model.FileInfo{Hash: string(file.Hashes.Sha1), DownloadLink: file.Downloads}
	}

	// Add overrides file
	r, err := zip.OpenReader(modPackPatch)
	if err != nil {
		return nil, err
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

		filePath := f.Name
		if strings.HasPrefix(filePath, "overrides/") {
			filePath = strings.TrimPrefix(filePath, "overrides/")
		} else if strings.HasPrefix(filePath, "server-overrides/") {
			filePath = strings.TrimPrefix(filePath, "server-overrides/")
		} else {
			continue
		}

		readCloser, err := f.Open()
		if err != nil {
			return nil, err
		}

		fileHash, err := util.GetReadCloserSha1(readCloser)
		if err != nil {
			return nil, err
		}
		err = readCloser.Close()
		if err != nil {
			return nil, err
		}
		modPackInfo.File[model.Path(filePath)] = model.FileInfo{Hash: fileHash}
	}

	return &modPackInfo, nil
}

func CompareModPackInfo(oldVersion model.ModPackInfo, newVersion model.ModPackInfo) (deleteFileInfo *model.ModPackInfo, updateFileInfo *model.ModPackInfo, err error) {
	if oldVersion.ModPackName != newVersion.ModPackName || !reflect.DeepEqual(oldVersion.Dependencies, newVersion.Dependencies) {
		return nil, nil, errors.New("for mismatched versions, please upgrade manually")
	}

	for path := range oldVersion.File {
		if newVersion.File[path].Hash == oldVersion.File[path].Hash {
			delete(oldVersion.File, path)
			delete(newVersion.File, path)
		}
	}

	return &oldVersion, &newVersion, nil
}
