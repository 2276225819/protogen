package main

import (
	"archive/zip"
	"bytes"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
)

var url = "https://packages.grpc.io/archive/2024/03/c910004328210668e0180847c35f9d2e82fa81dd-f88f5a84-a5e1-440a-8465-d9ef99a01bc1/protoc/grpc-protoc_windows_x64-1.63.0-dev.zip"

func loadfile(tmpDir string) error {

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "打开文件失败:")

	}
	defer resp.Body.Close()

	// 解压文件
	b, _ := io.ReadAll(resp.Body)
	zipFile, err := zip.NewReader(bytes.NewReader(b), resp.ContentLength)
	if err != nil {
		return errors.Wrap(err, "解压文件失败:")

	}
	regrn, err := regexp.Compile("^grpc_(.+)_plugin\\.exe$")
	if err != nil {
		return errors.Wrap(err, "解压文件失败:")
	}

	for _, file := range zipFile.File {
		filePath := path.Join(tmpDir, file.Name)
		if regrn.MatchString(file.Name) {
			filePath = path.Join(tmpDir, regrn.ReplaceAllString(file.Name, "protoc-gen-${1}-grpc.exe"))
		}
		if file.FileInfo().IsDir() {
			// 创建目录
			os.MkdirAll(filePath, file.Mode())
		} else {
			// 创建文件
			writer, err := os.Create(filePath)
			if err != nil {
				return errors.Wrap(err, "创建文件失败:")

			}
			defer writer.Close()

			// 复制文件内容
			reader, err := file.Open()
			if err != nil {
				return errors.Wrap(err, "打开文件失败:")

			}
			defer reader.Close()

			_, err = io.Copy(writer, reader)
			if err != nil {
				return errors.Wrap(err, "复制文件内容失败:")

			}
		}
	}

	return nil
}
