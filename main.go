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
	"strings"
)

var (
	proxyUrl          = os.Getenv("PROXY_URL")
	apiUrl            = os.Getenv("API_URL")
	field             = os.Getenv("FIELD")
	port              = os.Getenv("PORT")
	subdomain         = os.Getenv("SUBDOMAIN")
	selfDomain        = os.Getenv("SELF_DOMAIN")
	cdtAllowedDomains []string
)

func main() {
	if selfDomain == "" {
		selfDomain = "craftjobs.net"
	}

	cdtAllowedDomainsEnv := os.Getenv("CDT_ALLOWED_DOMAINS")

	if cdtAllowedDomainsEnv != "" {
		cdtAllowedDomains = strings.Split(cdtAllowedDomainsEnv, ",")
	}

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
	loginUrl := "https://craftjobs.net/i/gklogin?f=" + field + "&s="

	if selfDomain != "craftjobs.net" {
		loginUrl += "cdt&d=" + selfDomain
	} else {
		loginUrl += subdomain
	}

	w.Header().Set(
		"Location",
		loginUrl)
	w.WriteHeader(302)
}

func bye(w http.ResponseWriter) {
	w.Header().Set("Location", "https://craftjobs.net")
	w.WriteHeader(302)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if selfDomain == "cdt.craftjobs.net" {
			redirDomain := r.URL.Query().Get("d")

			if !contains(cdtAllowedDomains, redirDomain) {
				bye(w)
				return
			}

			token, err := r.Cookie("gktoken")

			if err != nil {
				// Would normally redirect() here, but bye() instead because it's cdt and you can
				// only mess this up if you're trying to exploit
				bye(w)
				return
			}

			w.Header().Set(
				"Location",
				"https://"+redirDomain+"/_gk/cdt?t="+token.Value)
			w.WriteHeader(302)
			return
		}

		if r.URL.Path == "/_gk/cdt" { // Cross-domain token
			// Don't allow cdt on craftjobs.net
			if selfDomain == "craftjobs.net" {
				bye(w)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:   "gktoken",
				Value:  r.URL.Query().Get("t"),
				Domain: "." + selfDomain,
				Path:   "/",
				MaxAge: 99999999,
			})

			w.Header().Set("Location", "https://"+selfDomain)
			w.WriteHeader(302)

			return
		}

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

		if !jsonResp["field"] {
			bye(w)
			return
		}

		p.ServeHTTP(w, r)
	}
}
