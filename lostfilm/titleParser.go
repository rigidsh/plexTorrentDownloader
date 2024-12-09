package lostfilm

import (
	"errors"
	"regexp"
	"strconv"
)

var regExp = regexp.MustCompile(`(?P<name>.+)\((?P<originName>.+)\)\.\s*(?P<episodeName>.+)\.\s*\(S(?P<season>[0-9]+)E(?P<episode>[0-9]+)\)`)

func parseTitle(name string) (ContentItem, error) {
	result := ContentItem{}

	if !regExp.MatchString(name) {
		return result, errors.New("invalid name")
	}

	match := regExp.FindStringSubmatch(name)
	var err error = nil

	for i, name := range regExp.SubexpNames() {
		switch name {
		case "name":
			result.Name = match[i]
		case "originName":
			result.OriginalName = match[i]
		case "episodeName":
			result.EpisodeName = match[i]
		case "season":
			var season int
			season, err = strconv.Atoi(match[i])
			result.Season = uint8(season)
		case "episode":
			var episode int
			episode, err = strconv.Atoi(match[i])
			result.Episode = uint8(episode)
		}
	}

	return result, err
}
