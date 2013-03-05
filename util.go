package extract

import (
	"strconv"
	"strings"
)

func GetSite(url string) string {
	scheme := "http://"
	i := strings.Index(url, scheme);
	if i != -1 {
		url = url[len(scheme):]
	}
	i = strings.Index(url, "/")
	if i != -1 {
		url = url[0:i]
	}
	return url
}

func GetDomain(url string) string {
	scheme := "http://"
	i := strings.Index(url, scheme);
	if i != -1 {
		url = url[len(scheme):]
	}
	i = strings.Index(url, "/")
	if i != -1 {
		url = url[0:i]
	}
	i = strings.LastIndex(url, ".")
	if i == -1 {
		return url
	}
	seg := url[0:i]
	i = strings.LastIndex(seg, ".")
	if i == -1 {
		return url
	}
	domain := url[i+1:]

	return domain
}

func MapAddArrayPanic(m map[string]string, a []string) {
	for i, s := range(a) {
		index := strconv.Itoa(i)
		_, ok := m[index]
		if ok {
			panic("[MapAddArray] " + index  + " already exist in map")
		}

		m[index] = s
	}
}

func MapAddArrayCover(m map[string]string, a []string) {
	for i, s := range(a) {
		index := strconv.Itoa(i)
		_, ok := m[index]
		if ok {
			delete(m, index)
		}
		m[index] = s
	}
}

func MapAddArrayDiscard(m map[string]string, a []string) {
	for i, s := range(a) {
		if _, ok := m[s]; !ok {
			m[strconv.Itoa(i)] = s
		}
	}
}
