package main

import (
	"bytes"
	"encoding/json"
	"flag"
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
	"time"

	"github.com/mrjones/oauth"
)

const (
	requestTokenURL   string = "https://api.twitter.com/oauth/request_token"
	authorizeTokenURL string = "https://api.twitter.com/oauth/authorize"
	accessTokenURL    string = "https://api.twitter.com/oauth/access_token"
	apiBase           string = "https://api.twitter.com/1.1/"
	apiGetFavorites   string = apiBase + "favorites/list.json?include_entities=true&count=200"
	apiUnFavorite     string = apiBase + "favorites/destroy.json?id="
	apiRequestLimit   int    = 75
)

var (
	tweetID string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&tweetID, "tweetid", "", "Get media with an ID less than (that is, older than) or equal to the specified ID.")
	flag.StringVar(&tweetID, "i", "", "Get media with an ID less than (that is, older than) or equal to the specified ID.")
	flag.Parse()
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
		var consumerKey, consumerSecret, dlPath, filterAccount, dlWithoutAsk, unFavAfterDL string
		logPath := "~/twtr_fav_img_dl.log"
		fmt.Print("Enter consumer key: ")
		fmt.Scanln(&consumerKey)
		fmt.Print("Enter consumer secret: ")
		fmt.Scanln(&consumerSecret)
		fmt.Print("Enter download path: ")
		fmt.Scanln(&dlPath)
		fmt.Print("Enter twitter screen name that want to filter for download (separate by comma): ")
		fmt.Scanln(&filterAccount)
		fmt.Print("Continue download without asking? (y/N): ")
		fmt.Scanln(&dlWithoutAsk)
		dlWithoutAsk = strings.ToLower(dlWithoutAsk)
		fmt.Print("Un-favorite tweet after download? (y/N): ")
		fmt.Scanln(&unFavAfterDL)
		unFavAfterDL = strings.ToLower(unFavAfterDL)
		fmt.Print("Enter log path: ")
		fmt.Scanln(&logPath)

		cfg["ConsumerKey"] = consumerKey
		cfg["ConsumerSecret"] = consumerSecret
		cfg["DownloadPath"] = dlPath
		cfg["FilterAccount"] = filterAccount
		cfg["LogPath"] = logPath

		if dlWithoutAsk == "y" || dlWithoutAsk == "yes" {
			cfg["DownloadWithoutAsking"] = "true"
		} else {
			cfg["DownloadWithoutAsking"] = "false"
		}

		if unFavAfterDL == "y" || unFavAfterDL == "yes" {
			cfg["UnFavAfterDownload"] = "true"
		} else {
			cfg["UnFavAfterDownload"] = "false"
		}
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

func unFavoriteTweet(wg *sync.WaitGroup, client *http.Client, entityID string) {
	defer wg.Done()

	url := apiUnFavorite + entityID

	reqBody := map[string]string{"id": entityID}
	jsonBody, _ := json.Marshal(reqBody)
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Println(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("%v: %v\n", url, resp.Status)
	}
}

func downloadWorker(wg *sync.WaitGroup, url string, dlPath string, fileName string) {
	defer wg.Done()

	log.Printf("Get %v\n", fileName)

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()

	if err := os.MkdirAll(dlPath, 0755); err != nil {
		log.Println(err)
	}

	f, err := os.Create(filepath.Join(dlPath, fileName))
	if err != nil {
		log.Println(err)
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		log.Println(err)
	}

	f.Close()
}

func downloadMedia(client *http.Client, url string, dlPath string, filterAccount []string, unFav bool, logPath string) (string, error) {
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

	var fav []FavoriteList

	if err := json.Unmarshal([]byte(body), &fav); err != nil {
		return "", err
	}

	var lastTweetID string
	var wg sync.WaitGroup

	for _, v := range fav {
		i := sort.SearchStrings(filterAccount, strings.ToLower(v.User.ScreenName))

		if (i < len(filterAccount) && filterAccount[i] == strings.ToLower(v.User.ScreenName)) || len(filterAccount) <= 0 {
			for _, val := range v.Entities.Media {
				largeMediaURL := val.MediaURL + ":large"
				sl := strings.Split(val.MediaURL, "/")
				fileName := sl[len(sl)-1]

				wg.Add(1)
				go downloadWorker(&wg, largeMediaURL, filepath.Join(dlPath, v.User.ScreenName), fileName)
			}

			for _, val := range v.ExtendedEntities.Media {
				largeMediaURL := val.MediaURL + ":large"
				sl := strings.Split(val.MediaURL, "/")
				fileName := sl[len(sl)-1]

				wg.Add(1)
				go downloadWorker(&wg, largeMediaURL, filepath.Join(dlPath, v.User.ScreenName), fileName)
			}

			if unFav {
				if len(v.Entities.Media) == 0 && len(v.ExtendedEntities.Media) == 0 {
					f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						fmt.Println(err)
						continue
					}
					defer f.Close()

					for _, val := range v.Entities.Urls {
						_, _ = f.WriteString(val.URL + "\n")
					}
				}

				wg.Add(1)
				go unFavoriteTweet(&wg, client, v.IDStr)
			}

			log.Printf("ScreenName: %v, Tweet ID: %v, Num of media: %v\n", v.User.ScreenName, v.IDStr, len(v.Entities.Media)+len(v.ExtendedEntities.Media))
		}

		lastTweetID = v.IDStr
	}

	wg.Wait()

	return lastTweetID, nil
}

func main() {
	cfgFile, cfg, err := getConfig()
	if err != nil {
		log.Fatalln("Get configuration file failed: ", err)
	}

	var filterAccount []string
	if len(cfg["FilterAccount"]) > 0 {
		fmt.Printf("Filter account: %v\n", cfg["FilterAccount"])

		filterAccount = strings.Split(strings.ToLower(cfg["FilterAccount"]), ",")
		sort.Strings(filterAccount)
	}

	var dlWithoutAsk bool
	if strings.ToLower(cfg["DownloadWithoutAsking"]) == "true" {
		dlWithoutAsk = true
	} else {
		dlWithoutAsk = false
	}

	var unFav bool
	if strings.ToLower(cfg["UnFavAfterDownload"]) == "true" {
		unFav = true
	} else {
		unFav = false
	}

	logPath, foundLogPath := cfg["LogPath"]
	if !foundLogPath {
		currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalln(err)
		}

		logPath = filepath.Join(currentPath, "twtr_fav_img_dl.log")
	}
	if logPath[:2] == "~/" {
		logPath = filepath.Join(getHomeDir(), logPath[2:])
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
		log.Fatalln(err)
	}

	cfg["AccessToken"] = authorizeToken.Token
	cfg["AccessSecret"] = authorizeToken.Secret
	err = writeConfig(cfg, cfgFile)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := c.MakeHttpClient(authorizeToken)
	if err != nil {
		log.Fatalln(err)
	}

	dlPath, foundDLPath := cfg["DownloadPath"]
	if !foundDLPath {
		currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalln(err)
		}

		dlPath = filepath.Join(currentPath, "downloads")
	}
	if dlPath[:2] == "~/" {
		dlPath = filepath.Join(getHomeDir(), dlPath[2:])
	}
	dlPath = filepath.Join(dlPath, "twitter-favorite-pics")

	var lastTweetID string
	if tweetID != "" {
		lastTweetID = tweetID
		fmt.Printf("Specify tweet ID: %v\n", tweetID)

	}

	continueDL := "y"
	reqCounter := 0
	for continueDL == "y" || dlWithoutAsk {
		if reqCounter >= apiRequestLimit {
			log.Printf("Last tweet ID: %v\n", lastTweetID)
			log.Println("API request quota exceeded, wait for 16 minute to unlock...")
			time.Sleep(16 * time.Minute)
			reqCounter = 0
		}

		url := apiGetFavorites
		if lastTweetID != "" {
			url = apiGetFavorites + "&max_id=" + lastTweetID
		}

		prevTweetID, err := downloadMedia(client, url, dlPath, filterAccount, unFav, logPath)
		if err != nil {
			log.Fatalln(err)
			break
		}

		if prevTweetID > lastTweetID {
			log.Fatalf("No older tweet, exit! Last tweet ID: %v\n", prevTweetID)
		} else {
			lastTweetID = prevTweetID
		}

		if !dlWithoutAsk {
			fmt.Print("Type 'y' to continue: ")
			fmt.Scanln(&continueDL)
		}

		reqCounter++
	}

	log.Printf("Last tweet ID: %v\n", lastTweetID)
	log.Printf("All media is stored in: %v\n", dlPath)
}
