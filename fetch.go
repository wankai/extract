// features:
// routine安全
// 定制user agent
// 支持代理
// 支持静态Cookie
// 支持重定向

package extract

import (
	"os"
	"path/filepath"
	"errors"
	"io"
	"bufio"
	"strings"
	"net/http"
	"net/url"
	"sync"
)

var defaultUserAgent = "fetch"

type ProxyList struct {
	list []string
	index int
	mutex sync.Mutex
}

var defaultProxyList = &ProxyList{}

func (pl *ProxyList) load(file string) (err error) {
	var f *os.File
	var line string

	if f, err = os.Open(file); err != nil {
		return err
	}
	r := bufio.NewReader(f)
	for {
		if line, err = r.ReadString('\n'); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		line = strings.TrimSpace(line)
		pl.list = append(pl.list, line)
	}
	return nil
}

func (pl *ProxyList) getProxy() (u *url.URL, err error) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	if len(pl.list) == 0 {
		return nil, nil
	}
	s := pl.list[0]
	pl.index++
	if pl.index >= len(pl.list) {
		pl.index = 0
	}
	return url.Parse(s)
}

func proxyFunc(req* http.Request)(*url.URL, error) {
	return defaultProxyList.getProxy()
}

type CookieSet struct {
	data map[string] []*http.Cookie
}

func (cs *CookieSet) SetCookies(u *url.URL, cookies []*http.Cookie) {
	domain := GetDomain(u.Host)
	cs.data[domain] = cookies
}

func (cs *CookieSet) Cookies(u *url.URL) []*http.Cookie {
	domain := GetDomain(u.Host)
	return cs.data[domain]
}

func (cs *CookieSet) load(dir string) error {
	var err error

	loadOne := func(dir string, info os.FileInfo, e error) error {
		var f *os.File
		var err error
		var line string

		if info.IsDir() {
			return nil
		}
		file := dir + "/" + info.Name()
		if f, err = os.Open(file); err != nil {
			return err
		}
		r := bufio.NewReader(f)
		for {
			if line, err = r.ReadString('\n'); err != nil {
				if err == io.EOF {
					break
				}
				i := strings.Index(line, "=")
				if i == -1 {
					return errors.New(file + " format error")
				}
				name := strings.TrimSpace(line[0:i])
				value := strings.TrimSpace(line[i+1:])
				if name == "" || value == "" {
					return errors.New(file + " format error")
				}
				domain := filepath.Base(dir)
				if _, ok := cs.data[domain]; !ok {
					cs.data[domain] = make([]*http.Cookie, 0)
				}
				cs.data[domain] = append(cs.data[domain], &http.Cookie{Name: name, Value: value})
			}
		}
		return nil
	}

	if err = filepath.Walk(dir, loadOne); err != nil {
		return err
	}
	return nil
}

var defaultCookieSet = &CookieSet{ data: make(map[string] []*http.Cookie) }

type Client struct {
	http.Client
	UserAgent string
	ProxyFile string
	CookieDir string
}

func (c *Client) Init() (err error) {
	if c.ProxyFile != "" {
		if err = defaultProxyList.load(c.ProxyFile); err != nil {
			return err
		}
	}
	if c.CookieDir != "" {
		if err = defaultCookieSet.load(c.CookieDir); err != nil {
			return err
		}
	}
	c.Transport = &http.Transport{Proxy: proxyFunc}
	c.Jar =  defaultCookieSet

	if c.UserAgent == "" {
		c.UserAgent = "zwk fetch"
	}
	return nil
}

func (c *Client) GetReferer(url string, referer string) (res *http.Response, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	}
	if len(referer) > 0 {
		req.Header.Set("Referer", referer)
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return c.Do(req)
}
