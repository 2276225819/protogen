package main

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var C = "config.yaml"
var T = false
var cfg = struct {
	Paths   []string `yaml:"proto_paths"`
	Files   []string `yaml:"proto_files"`
	Plugins []struct {
		Name string `yaml:"name"`
		Out  string `yaml:"out"`
		Opt  any    `yaml:"opt"`
	} `yaml:"plugins"`
	Option map[string]string `yaml:"option"`
}{}

func main() {
	flag.StringVar(&C, "c", C, "config path")
	flag.BoolVar(&T, "t", false, "test run")
	flag.Parse()

	exe := "protoc "
	txt, e := os.ReadFile(C)
	if e != nil {
		tx := "# @see https://github.com/2276225819/protogen/blob/master/example.config.yaml\n\n"+
			"option: #文件选项\n" +
			"  # go_package: \"GrpcService1/\"\n" +
			"plugins: #插件和输出路径\n" +
			"  # - name: go \n" +
			"  #   out: \"./go\" \n" +
			"  #   opt: \"./go\" \n" +
			"proto_paths: #引用路径\n" +
			"  # - \"./protoc\"\n" +
			"proto_files: #输入路径\n" +
			"  # - \"./protoc/*.proto\"\n"
		_ = os.WriteFile(C, []byte(tx), os.FileMode(0777))
		panic(errors.Wrap(e, "找不到配置文件，已重新生成 "+C))
	}
	ee := yaml.Unmarshal(txt, &cfg)
	if ee != nil {
		panic(errors.Wrap(e, "配置解析失败"))
	}
	_, e = bash(exe + " --version")
	if e != nil {
		exe = "./protoc "
		_, e = bash(exe + " --version")
		if e != nil {
			fmt.Println("找不到 protoc 官网下载中...")
			e := loadfile()
			if ee != nil {
				panic(errors.Wrap(e, "下载失败 需要到官网下载: \\n https://packages.grpc.io/archive/2019/12/e522302e33b2420722f866e3de815e4e0a1d9952-219973fd-1007-4db7-a78f-976ec554952d/index.xml"))
			}
		}
	}

	// 补全 option
	Opts := []string{}
	for k := range cfg.Option {
		Opts = append(Opts, k)
	}
	Defers := map[string][2]func(){}
	pkgGoPackage := regexp.MustCompile("[\r\n]package\\s+([^\\s;]+)")
	optGoPackage := regexp.MustCompile("[\r\n]option\\s+(\\S+)\\s*=[^\r\n]+")
	for _, _path := range ls(cfg.Files) {
		path := _path
		_txt, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		txt := string(_txt)
		eOpt := []string{}
		for _, i2 := range optGoPackage.FindAllStringSubmatch(txt, -1) {
			eOpt = append(eOpt, i2[1])
		}
		pkg := pkgGoPackage.FindStringSubmatch(txt)
		for _, ff := range diff(Opts, eOpt) {
			val := cfg.Option[ff]
			if ff == "go_package" && len(pkg) == 2 {
				val = cfg.Option[ff] + "/" + pkg[1]
			}
			txt += "\noption " + ff + " = \"" + val + "\"; \n"
		}
		Defers[path] = [2]func(){
			func() { _ = os.WriteFile(path, []byte(txt), os.FileMode(0777)) },
			func() { _ = os.WriteFile(path, _txt, os.FileMode(0777)) },
		}
		fmt.Println("add option " + path)
	}

	// 拼接命令
	for _, path := range cfg.Paths {
		exe += "-I=" + (path) + " "
	}
	for _, c := range cfg.Plugins {
		exe += "--" + c.Name + "_out=" + c.Out + " "
		switch f := c.Opt.(type) {
		case []any:
			for _, s := range f {
				exe += "--" + c.Name + "_opt=" + s.(string) + " "
			}
		case string:
			exe += "--" + c.Name + "_opt=" + f + " "
		}
	}
	exe += "\t" + strings.Join(cfg.Files, " ")

	// 执行命令
	if !T {
		for _, _txt := range Defers {
			_txt[0]()
		}
		defer func() {
			for _, _txt := range Defers {
				_txt[1]()
			}
		}()
		_, err := bash(exe)
		if err != nil {
			panic(err)
		}
		fmt.Println("done")
	} else {
		exe = strings.ReplaceAll(exe, "--", "\n--")
		exe = strings.ReplaceAll(exe, "\t", "\n  ")
		fmt.Println(exe)
	}
}

func diff[T comparable](b []T, bb []T) (n []T) {
	for _, t := range b {
		f := true
		for _, t2 := range bb {
			if t2 == t {
				f = false
				break
			}
		}
		if !f {
			continue
		}
		n = append(n, t)
	}
	return
}

func ls(path []string) (ss []string) {
	for _, v := range path {
		vv, _ := filepath.Glob(v)
		ss = append(ss, vv...)
	}
	return
}

func bash(a ...string) (string, error) {
	s := make([]string, 0, 4)
	if filepath.Separator == '/' {
		s = append(s, "sh", "-c")
		s = append(s, strings.Join(a, "\n"))
	} else {
		s = append(s, "cmd", "/A", "/C")
		a = append([]string{"chcp 65001"}, a...) //windows
		for k, s2 := range a {
			/* https://blog.csdn.net/kucece/article/details/46716069 */
			s2 = strings.ReplaceAll(s2, "^", "^^")
			s2 = strings.ReplaceAll(s2, "&", "^&")
			s2 = strings.ReplaceAll(s2, ">", "^>")
			s2 = strings.ReplaceAll(s2, "|", "^|")
			a[k] = s2
		}
		str := strings.Join(a, " & ")
		s = append(s, str)
	}
	cmd := exec.Command(s[0], s[1:]...)
	cmd.Dir = os.TempDir()
	out, e := cmd.CombinedOutput()
	if e != nil {
		return "", errors.New("[bash]\n" + strings.Join(s, " ") + "\n" + (string)(out) + "\n" + e.Error())
	}
	return string(out), nil
}
