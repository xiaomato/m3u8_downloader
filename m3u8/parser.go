package m3u8

import (
	"io/ioutil"
	"net/http"
	"strings"
)

// #EXT-X-VERSION:3
// #EXT-X-TARGETDURATION:10
// #EXT-X-KEY:METHOD=AES-128,URI="https://p.sxjychjs.com/watch3/48d57558d55f0e4e7d23bad0f7125929/crypt.key?auth_key=1655718954-0-0-e4a54d773c7fc0cf39ecd940d07932f9"
// #EXT-X-MEDIA-SEQUENCE:0

func ParseURL(m3u8 string) ([]string, map[string]string, error) {
	rsp, err := http.Get(m3u8)
	if err != nil {
		return nil, nil, err
	}
	text, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, nil, err
	}
	links, infos := ParseText(string(text))
	if infos["URI"] == "" {
		return links, infos, nil
	}
	if !strings.HasPrefix(infos["URI"], "http") {
		parts := strings.Split(links[0], "/")
		parts[len(parts)-1] = infos["URI"]
		infos["URI"] = strings.Join(parts, "/")
	}
	rsp, err = http.Get(infos["URI"])
	if err != nil {
		return nil, nil, err
	}
	text, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, nil, err
	}
	infos["KEY"] = string(text)
	return links, infos, nil
}

func ParseText(text string) ([]string, map[string]string) {
	lines := strings.Split(text, "\n")
	var links []string
	var values = map[string]string{}
	for _, l := range lines {
		if strings.HasPrefix(l, "#") {
			for k, v := range ParseNoteLine(l) {
				values[k] = v
			}
		}
		if strings.HasPrefix(l, "http") {
			links = append(links, l)
		}
	}
	return links, values
}

func ParseNoteLine(line string) map[string]string {
	if strings.Contains(line, ":") {
		i := strings.Index(line, ":")
		key, value := line[:i], line[i+1:]
		if strings.Contains(value, "=") {
			return ParseKeyValue(value)
		}
		return map[string]string{key: value}
	}
	return nil
}

func ParseKeyValue(value string) map[string]string {
	value = strings.ReplaceAll(value, "\"", "")
	res := make(map[string]string)
	parts := strings.Split(value, ",")
	for _, p := range parts {
		if strings.Contains(p, "=") {
			i := strings.Index(p, "=")
			res[p[:i]] = p[i+1:]
		}
	}
	return res
}
