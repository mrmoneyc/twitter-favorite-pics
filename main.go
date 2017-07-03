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
	requestTokenURL   string = "https://api.twitter.com/oauth/request_token"
	authorizeTokenURL string = "https://api.twitter.com/oauth/authorize"
	accessTokenURL    string = "https://api.twitter.com/oauth/access_token"
	apiBase           string = "https://api.twitter.com/1.1/"
	apiGetFavorites   string = apiBase + "favorites/list.json?count=200"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func getHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}

		return home
	}

	return os.Getenv("HOME")
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
		var consumerKey, consumerSecret, dlPath, filterAccount string
		fmt.Print("Enter consumer key: ")
		fmt.Scanln(&consumerKey)
		fmt.Print("Enter consumer secret: ")
		fmt.Scanln(&consumerSecret)
		fmt.Print("Enter download path: ")
		fmt.Scanln(&dlPath)
		fmt.Print("Enter twitter screen name that want to filter for download (separate by comma): ")
		fmt.Scanln(&filterAccount)

		cfg["ConsumerKey"] = consumerKey
		cfg["ConsumerSecret"] = consumerSecret
		cfg["DownloadPath"] = dlPath
		cfg["FilterAccount"] = filterAccount
	} else {
		err = json.Unmarshal(buf, &cfg)
		if err != nil {
			return "", nil, fmt.Errorf("Could not unmarshal %v: %v", cfgFile, err)
		}
	}

	return cfgFile, cfg, nil
}

func writeConfig(cfg map[string]string, cfgFile string) error {
	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(cfgFile, buf, 0700)
	if err != nil {
		return err
	}

	return nil
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

func downloadWorker(wg *sync.WaitGroup, url string, dlPath string, fileName string) {
	defer wg.Done()

	fmt.Printf("Get: %v\n", fileName)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if err := os.MkdirAll(dlPath, 0755); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(filepath.Join(dlPath, fileName))
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
}

func downloadMedia(client *http.Client, url string, dlPath string, filterAccount []string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(resp.Status)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var fav FavoriteList

	if err := json.Unmarshal([]byte(body), &fav); err != nil {
		return "", err
	}

	var wg sync.WaitGroup
	var lastTweetID string

	for _, v := range fav {
		i := sort.SearchStrings(filterAccount, v.User.ScreenName)

		if (i < len(filterAccount) && filterAccount[i] == v.User.ScreenName) || len(filterAccount) <= 0 {
			for _, val := range v.Entities.Media {
				largeMediaURL := val.MediaURL + ":large"
				sl := strings.Split(val.MediaURL, "/")
				fileName := sl[len(sl)-1]

				wg.Add(1)
				go downloadWorker(&wg, largeMediaURL, filepath.Join(dlPath, v.User.ScreenName), fileName)
			}

		}

		lastTweetID = v.IDStr
	}

	wg.Wait()

	return lastTweetID, nil
}

func main() {
	cfgFile, cfg, err := getConfig()
	if err != nil {
		log.Fatal("Get configuration file failed: ", err)
	}

	var filterAccount []string
	if len(cfg["FilterAccount"]) > 0 {
		fmt.Printf("Filter account: %v\n", cfg["FilterAccount"])

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
	err = writeConfig(cfg, cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	client, err := c.MakeHttpClient(authorizeToken)
	if err != nil {
		log.Fatal(err)
	}

	dlPath, foundDLPath := cfg["DownloadPath"]
	if !foundDLPath {
		currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatal(err)
		}

		dlPath = filepath.Join(currentPath, "downloads")
	}
	if dlPath[:2] == "~/" {
		dlPath = filepath.Join(getHomeDir(), dlPath[2:])
	}
	dlPath = filepath.Join(dlPath, "twitter-favorite-pics")

	var lastTweetID string
	continueDL := "y"
	for continueDL == "y" {
		url := apiGetFavorites
		if lastTweetID != "" {
			url = apiGetFavorites + "&max_id=" + lastTweetID
		}

		lastTweetID, err = downloadMedia(client, url, dlPath, filterAccount)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print("Type 'y' to continue: ")
		fmt.Scanln(&continueDL)
	}

	fmt.Printf("All media is stored in: %v\n", dlPath)
}
