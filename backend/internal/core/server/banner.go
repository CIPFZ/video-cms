package server

import "fmt"

func PrintBanner(port int) {
	fmt.Printf(`
🚀 欢迎使用影视 CMS
📦 当前版本: v0.0.1
📜 文档地址: http://127.0.0.1:%d/swagger/index.html
`, port)
}
