//go:build !(hik && cgo)

// This stub keeps `go build ./...` working on machines without the Hikvision
// HCNetSDK. The real implementation is compiled with `-tags hik` and cgo.
package main

import "fmt"

func main() {
	fmt.Println("hik-ipc-plugin: build with `-tags hik` and the HCNetSDK present under ./sdk")
	fmt.Println("see tools/hik-ipc-plugin/README.md")
}
