package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/mrjones/oauth"
)

const (
	consumerKey       string = ""
	consumerSecret    string = ""
	requestTokenURL   string = "https://api.twitter.com/oauth/request_token"
	authorizeTokenURL string = "https://api.twitter.com/oauth/authorize"
	accessTokenURL    string = "https://api.twitter.com/oauth/access_token"
	apiBase           string = "https://api.twitter.com/1.1/"
	apiFavoriteList   string = apiBase + "favorites/list.json?count=201"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func getConfig() (string, map[string]string, error) {
	cfgFolder := os.Getenv("HOME")

	if cfgFolder == "" && runtime.GOOS == "windows" {
		cfgFolder = os.Getenv("APPDATA")

		if cfgFolder == "" {
			cfgFolder = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", "twitter-favorite-pics")
		}

		cfgFolder = filepath.Join(cfgFolder, "twitter-favorite-pics")
	} else {
		cfgFolder = filepath.Join(cfgFolder, ".config", "twitter-favorite-pics")
	}

	if err := os.MkdirAll(cfgFolder, 0700); err != nil {
		return "", nil, err
	}

	cfgFile := filepath.Join(cfgFolder, "settings.json")

	cfg := map[string]string{}

	buf, err := ioutil.ReadFile(cfgFile)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}

	if err != nil {
		cfg["ConsumerKey"] = consumerKey
		cfg["ConsumerSecret"] = consumerSecret
		cfg["FilterAccount"] = ""
	} else {
		err = json.Unmarshal(buf, &cfg)
		if err != nil {
			return "", nil, fmt.Errorf("Could not unmarshal %v: %v", cfgFile, err)
		}
	}

	return cfgFile, cfg, nil
}

func writeConfig(cfg map[string]string, cfgFile string) (bool, error) {
	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}

	err = ioutil.WriteFile(cfgFile, buf, 0700)
	if err != nil {
		return false, err
	}

	return true, nil
}

func openBrowser(url string) error {
	browser := "xdg-open"
	args := []string{url}

	if runtime.GOOS == "windows" {
		browser = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", url}
	} else if runtime.GOOS == "darwin" {
		browser = "open"
	}

	browser, err := exec.LookPath(browser)
	if err != nil {
		return err
	}

	cmd := exec.Command(browser, args...)
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func getAuthorizeToken(c *oauth.Consumer, cfg map[string]string) (*oauth.AccessToken, error) {
	accessToken, foundToken := cfg["AccessToken"]
	accessSecret, foundSecret := cfg["AccessSecret"]

	var authorizeToken *oauth.AccessToken

	if foundToken && foundSecret {
		authorizeToken = &oauth.AccessToken{Token: accessToken, Secret: accessSecret}
	} else {
		reqToken, url, err := c.GetRequestTokenAndUrl("")
		if err != nil {
			return nil, err
		}

		fmt.Println("(1) Go to: " + url)
		fmt.Println("(2) Grant access, you should get back a verification code.")
		fmt.Print("(3) Enter that verification code here: ")

		err = openBrowser(url)
		if err != nil {
			return nil, err
		}

		verificationCode := ""
		fmt.Scanln(&verificationCode)

		authorizeToken, err = c.AuthorizeToken(reqToken, verificationCode)
		if err != nil {
			return nil, err
		}
	}

	return authorizeToken, nil
}

func downloadMedia(wg *sync.WaitGroup, url string, fileName string, screenName string) {
	defer wg.Done()
	fmt.Printf("Get %v\n", fileName)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	downloadPath := filepath.Join(currentPath, "downloads", screenName)
	if err := os.MkdirAll(downloadPath, 0755); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(filepath.Join(downloadPath, fileName))
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
}

func main() {
	cfgFile, cfg, err := getConfig()
	if err != nil {
		log.Fatal("Get configuration file failed: ", err)
	}

	var filterAccount []string
	if len(cfg["FilterAccount"]) > 0 {
		filterAccount = strings.Split(cfg["FilterAccount"], ",")
		sort.Strings(filterAccount)
	}

	c := oauth.NewConsumer(
		cfg["ConsumerKey"],
		cfg["ConsumerSecret"],
		oauth.ServiceProvider{
			RequestTokenUrl:   requestTokenURL,
			AuthorizeTokenUrl: authorizeTokenURL,
			AccessTokenUrl:    accessTokenURL,
		},
	)

	authorizeToken, err := getAuthorizeToken(c, cfg)
	if err != nil {
		log.Fatal(err)
	}

	cfg["AccessToken"] = authorizeToken.Token
	cfg["AccessSecret"] = authorizeToken.Secret
	_, err = writeConfig(cfg, cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	client, err := c.MakeHttpClient(authorizeToken)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Get(apiFavoriteList)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var fav FavoriteList

	if err := json.Unmarshal([]byte(body), &fav); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	for _, v := range fav {
		i := sort.SearchStrings(filterAccount, v.User.ScreenName)

		if i < len(filterAccount) && filterAccount[i] == v.User.ScreenName {
			for _, val := range v.Entities.Media {
				// fmt.Printf("[%v] media_url: %v\n", k, val.MediaURL)
				largeMediaURL := val.MediaURL + ":large"
				sl := strings.Split(val.MediaURL, "/")
				fileName := sl[len(sl)-1]
				// fmt.Printf("%v -> %v, %v\n", k, largeMediaURL, fileName)

				wg.Add(1)
				go downloadMedia(&wg, largeMediaURL, fileName, v.User.ScreenName)
			}
		}
	}

	wg.Wait()
	fmt.Println("Done")
}
