package main

import (
	"bufio"
	"fmt"
	"net/http"
)

func main() {
	resp, err := http.Get("http://127.0.0.1:8889/api/v1/namespaces/default/pods/jupyter-notebook-7dcf66bf99-5qpnr/log?container=jupyter-notebook&follow=true")
	if err != nil {
		panic(err)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			panic(err)
		}

		fmt.Println(string(line))
	}
}
