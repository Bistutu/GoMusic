package utils

import (
	"net/url"

	"GoMusic/initialize/log"
)

func GetSongsId(link string) (string, error) {
	parse, err := url.ParseRequestURI(link)
	if err != nil {
		log.Errorf("fail to parse url: %v", err)
		return "", err
	}
	query, err := url.ParseQuery(parse.RawQuery)
	return query.Get("id"), nil
}
