package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

var (
	proxyUrl = os.Getenv("PROXY_URL")
	apiUrl   = os.Getenv("API_URL")
	field    = os.Getenv("FIELD")
	port     = os.Getenv("PORT")
)

func main() {
	remote, err := url.Parse(proxyUrl)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

func redirect(w http.ResponseWriter) {
	w.Header().Set("Location", "https://craftjobs.net/i/gklogin?s="+field)
	w.WriteHeader(302)
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, err := r.Cookie("gktoken")

		if err != nil {
			fmt.Println("cookie error")
			redirect(w)
			return
		}

		bodyMap := make(map[string]string)

		bodyMap["token"] = auth.Value
		bodyMap["field"] = field

		bodyStr, err := json.Marshal(bodyMap)
		req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(bodyStr))

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)

		if err != nil {
			fmt.Println("bad err")
			fmt.Println("BAD THINGS HAVE HAPPENED!")
			fmt.Println(err)
			redirect(w)
			return
		}

		defer resp.Body.Close()

		jsonResp := make(map[string]bool)
		bodyRaw, _ := ioutil.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyRaw, &jsonResp)

		fmt.Println(string(bodyRaw))

		if !jsonResp["valid"] {
			fmt.Println("valid error")
			redirect(w)
			return
		}

		p.ServeHTTP(w, r)
	}
}
