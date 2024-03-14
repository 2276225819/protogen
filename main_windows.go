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

var url = "https://packages.grpc.io/archive/2019/12/e522302e33b2420722f866e3de815e4e0a1d9952-219973fd-1007-4db7-a78f-976ec554952d/protoc/grpc-protoc_windows_x64-1.27.0-dev.zip"

func loadfile() error {
	tmpDir := os.TempDir()

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
			filePath = path.Join(tmpDir, regrn.ReplaceAllString(file.Name, "protoc-gen-${1}.exe"))
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
