package main

// FavoriteList is mapping to user favorited tweet list
type FavoriteList struct {
	Coordinates        interface{} `json:"coordinates"`
	Truncated          bool        `json:"truncated"`
	Favorited          bool        `json:"favorited"`
	CreatedAt          string      `json:"created_at"`
	IDStr              string      `json:"id_str"`
	InReplyToUserIDStr interface{} `json:"in_reply_to_user_id_str"`
	Entities           struct {
		Urls []struct {
			URL         string `json:"url"`
			DisplayURL  string `json:"display_url"`
			ExpandedURL string `json:"expanded_url"`
		} `json:"urls"`
		HashTags     []interface{} `json:"hashtags"`
		UserMentions []interface{} `json:"user_mentions"`
		Media        []struct {
			ID            int    `json:"id"`
			IDStr         string `json:"id_str"`
			Indice        []int  `json:"indices"`
			MediaURL      string `json:"media_url"`
			MediaURLHTTPS string `json:"media_url_https"`
			URL           string `json:"url"`
			DisplayURL    string `json:"display_url"`
			ExpandedURL   string `json:"expanded_url"`
			Type          string `json:"type"`
			Sizes         struct {
				Medium struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"medium"`
				Thumb struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"thumb"`
				Small struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"small"`
				Large struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"large"`
			} `json:"sizes"`
		} `json:"media"`
	} `json:"entities"`
	Text                 string      `json:"text"`
	Contributors         interface{} `json:"contributors"`
	ID                   int         `json:"id"`
	RetweetCount         int         `json:"retweet_count"`
	InReplyToStatusIDStr interface{} `json:"in_reply_to_status_id_str"`
	Geo                  interface{} `json:"geo"`
	Retweeted            bool        `json:"retweeted"`
	InReplyToUserID      interface{} `json:"in_reply_to_user_id"`
	InReplyToScreenName  interface{} `json:"in_reply_to_screen_name"`
	Source               string      `json:"source"`
	User                 struct {
		ProfileSidebarFillColor   string `json:"profile_sidebar_fill_color"`
		ProfileBackgroundTile     bool   `json:"profile_background_tile"`
		ProfileSidebarBorderColor string `json:"profile_sidebar_border_color"`
		Name                      string `json:"name"`
		ProfileImageURL           string `json:"profile_image_url"`
		Location                  string `json:"location"`
		CreatedAt                 string `json:"created_at"`
		FollowRequestSent         bool   `json:"follow_request_sent"`
		IsTranslator              bool   `json:"is_translator"`
		IDStr                     string `json:"id_str"`
		ProfileLinkColor          string `json:"profile_link_color"`
		Entities                  struct {
			Description struct {
				Urls []interface{} `json:"urls"`
			} `json:"description"`
		} `json:"entities"`
		FavoritesCount                 int    `json:"favourites_count"`
		URL                            string `json:"url"`
		DefaultProfile                 bool   `json:"default_profile"`
		ContributorsEnabled            bool   `json:"contributors_enabled"`
		ProfileImageURLHTTPS           string `json:"profile_image_url_https"`
		UTCOffset                      int    `json:"utc_offset"`
		ID                             int    `json:"id"`
		ListedCount                    int    `json:"listed_count"`
		ProfileUseBackgroundImage      bool   `json:"profile_use_background_image"`
		FollowersCount                 int    `json:"followers_count"`
		Protected                      bool   `json:"protected"`
		ProfileTextColor               string `json:"profile_text_color"`
		Lang                           string `json:"lang"`
		ProfileBackgroundColor         string `json:"profile_background_color"`
		TimeZone                       string `json:"time_zone"`
		Verified                       bool   `json:"verified"`
		ProfileBackgroundImageURLHTTPS string `json:"profile_background_image_url_https"`
		Description                    string `json:"description"`
		GeoEnabled                     bool   `json:"geo_enabled"`
		Notifications                  bool   `json:"notifications"`
		DefaultProfileImage            bool   `json:"default_profile_image"`
		FriendsCount                   int    `json:"friends_count"`
		ProfileBackgroundImageURL      string `json:"profile_background_image_url"`
		StatusesCount                  int    `json:"statuses_count"`
		Following                      bool   `json:"following"`
		ScreenName                     string `json:"screen_name"`
		ShowAllInlineMedia             bool   `json:"show_all_inline_media"`
	} `json:"user"`
	Place             interface{} `json:"place"`
	InReplyToStatusID interface{} `json:"in_reply_to_status_id"`
}
