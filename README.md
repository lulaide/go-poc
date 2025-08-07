# go-poc

一个使用 Go 语言编写的长亭 POC 测试工具

## 添加子仓库

``` bash
git submodule add https://github.com/chaitin/xray.git xray
git submodule update --init --recursive
```

## 构建可执行文件

``` bash
go build
```

## 使用说明

- 指定POC 文件执行
  
```bash
./go-poc run --poc xray/pocs/apache-httpd-cve-2021-41773-path-traversal.yml --target http://localhost:8080
```

- 关键词搜索执行

```bash
./go-poc search --keyword apache --target http://localhost:8080
```

- 执行全部搜索到的 POC

```bash
./go-poc search --keyword apache --target http://localhost:8080 --all
./go-poc search --keyword apache --target http://localhost:8080 --all | grep "成功"
```
