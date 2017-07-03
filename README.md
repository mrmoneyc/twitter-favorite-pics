twitter-favorite-pics
======================
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/toomore/gogrs/master/LICENSE)

A twitter media download tool that supports concurrency download.
This tool help you to download images included in favorite tweet.

Install
--------------

    go get -u -x github.com/mrmoneyc/twitter-favorite-pics

You need go to [Twitter Application Management page](https://apps.twitter.com/) to get OAuth consumer key and secret.

Usage
---------------------

    twitter-favorite-pics

All the images will download to `CURRENT_WORKDIR/downloads` if you not set `DownloadPath` in `settings.json`.

License
---------------

MIT license
