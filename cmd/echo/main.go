package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"path"
	"sync"
)

const index = `
<!DOCTYPE html>
<html>
<head>
</head>
<body>
	<form action="/index.html" method="POST" enctype="multipart/form-data">
		<input name="title">
		<input name="author">
		<input name="attachment-file" type="file">
		<input type="submit">
	</form>
</body>
</html>
`

const redirectForm = `
<!DOCTYPE html>
<html>
<head></head>
<body>
	<form action="redirected-location" method="POST">
		<input type="hidden" name="data" value="message"/>
		<input type="submit" value="Continue" />
	</form>
</body>
</html>
`

func handler(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("%s\n", dump)

	switch path.Base(r.URL.Path) {
	case "favicon.ico":
		http.NotFound(w, r)
		return
	case "index.html":
		t, err := template.New("index.html").Parse(index)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, nil); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		return
	case "redirect-form":
		t, err := template.New("redirect-form.html").Parse(redirectForm)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, nil); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		return
	case "welcome":
		w.Header().Add("Set-Cookie", "VISIT=TRUE")
		if _, ok := r.Header["Cookie"]; ok {
			fmt.Fprintf(w, "<html><body>Thank you, comeback!</body></html>\n")
		} else {
			fmt.Fprintf(w, "<html><body>Thank you, you're first visit!</body></html>\n")
		}
	case "digest":
		fmt.Printf("URL: %s\n", r.URL.String())
		fmt.Printf("Query: %v\n", r.URL.Query())
		fmt.Printf("Proto: %s\n", r.Proto)
		fmt.Printf("Method: %s\n", r.Method)
		fmt.Printf("Header: %v\n", r.Header)
		defer r.Body.Close()

		body, _ := ioutil.ReadAll(r.Body)
		fmt.Printf("--body--\n%s\n", body)

		if _, ok := r.Header["Authorization"]; !ok {
			w.Header().Add("WWW-Authenticate",
				`Digest realm="Secret Zone", nonce="TgLc25U2BQA=f510a2780473e18e6587be8-2c2e78fe2b04afd", algorithm=MD5, qop="auth"`)
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			fmt.Fprintf(w, "<html><body>secret page</body></html>\n")
		}
	case "redirect-300":
		w.Header().Add("Location", "/redirected-location")
		w.WriteHeader(http.StatusMultipleChoices)
	case "redirect-301":
		w.Header().Add("Location", "/redirected-location")
		w.WriteHeader(http.StatusMovedPermanently)
	case "redirect-302":
		w.Header().Add("Location", "/redirected-location")
		w.WriteHeader(http.StatusFound)
	case "redirect-303":
		w.Header().Add("Location", "/redirected-location")
		w.WriteHeader(http.StatusSeeOther)
	case "redirect-307":
		w.Header().Add("Location", "/redirected-location")
		w.WriteHeader(http.StatusPermanentRedirect)
	}
	fmt.Fprintf(w, "<html><body>Hello, World!</body></html>\n")
}

func main() {
	sigCh := make(chan os.Signal)
	defer close(sigCh)

	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	var httpServer http.Server
	http.HandleFunc("/", handler)
	log.Println("start http listening :18888")
	httpServer.Addr = "127.0.0.1:18888"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, ok := <-sigCh
		if ok {
			if err := httpServer.Shutdown(context.Background()); err != nil {
				log.Fatalf("shutdown error: %v", err)
			}
			log.Printf("shutdown")
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("unexpected shutdown: %v", err)
	}
	wg.Wait()
	log.Printf("Bye!")
}
