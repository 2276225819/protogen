package main

import (
	"archive/tar"
	"compress/gzip"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

var url = "https://packages.grpc.io/archive/2024/03/c910004328210668e0180847c35f9d2e82fa81dd-f88f5a84-a5e1-440a-8465-d9ef99a01bc1/protoc/grpc-protoc_linux_x64-1.63.0-dev.tar.gz"

func loadfile() error {
	outputDir := os.TempDir()

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "打开文件失败:")

	}
	defer resp.Body.Close()

	// 解压 tar.gz 文件
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.Wrap(err, "解压文件失败:")

	}
	defer gzipReader.Close()

	regrn, err := regexp.Compile("^grpc_(.+)_plugin$")
	if err != nil {
		return errors.Wrap(err, "解压文件失败:")
	}
	tarReader := tar.NewReader(gzipReader)

	// 判断目录是否存在
	ExistDir := func(dirname string) bool {
		fi, err := os.Stat(dirname)
		return (err == nil || os.IsExist(err)) && fi.IsDir()
	}

	// 逐个解压文件
	for {
		header, err := tarReader.Next()
		if err != nil {
			break
		}
		outputFile := filepath.Join(outputDir, header.Name)
		// 根据 header 的 Typeflag 字段，判断文件的类型
		switch header.Typeflag {
		case tar.TypeDir: // 如果是目录时候，创建目录
			// 判断下目录是否存在，不存在就创建
			if b := ExistDir(outputFile); !b {
				// 使用 MkdirAll 不使用 Mkdir ，就类似 Linux 终端下的 mkdir -p，
				// 可以递归创建每一级目录
				if err := os.MkdirAll(outputFile, 0777); err != nil {
					return err
				}
			}
		case tar.TypeReg: // 如果是文件就写入到磁盘
			// 创建目标文件
			if regrn.MatchString(path.Base(header.Name)) {
				outputFile = path.Join(outputDir, regrn.ReplaceAllString(path.Base(header.Name), "protoc-gen-${1}-grpc"))
			}
			outFile, err := os.Create(outputFile)
			if err != nil {
				return errors.Wrap(err, "创建本地文件失败:")

			}
			defer outFile.Close()

			err = os.Chmod(outputFile, 0777)
			if err != nil {
				return errors.Wrap(err, "创建本地文件失败:")
			}

			// 将解压的内容写入目标文件
			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return errors.Wrap(err, "写入文件失败:")
			}
		}

	}

	return nil
}
