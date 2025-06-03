package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": // macOS
		err = exec.Command("open", url).Start()
	default: // Linux
		err = exec.Command("xdg-open", url).Start()
	}

	if err != nil {
		log.Printf("Falha ao abrir o navegador: %v\n", err)
	}
}

func main() {
	fileServer := http.FileServer(http.Dir("./static"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/login/login.html", http.StatusFound)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	url := "http://localhost:8081/"
	fmt.Println("Servidor rodando em", url)

	openBrowser(url)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}
