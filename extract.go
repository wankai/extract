package extract

import (
	"fmt"
	"os"
	"errors"
	"path/filepath"
	"net/http"
	"io/ioutil"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/html"
)

type FieldMethod func(string) string

type Options struct {
	TemplateDir string
	UserAgent string
	CookieDir string
	ProxyFile string
	Verbose bool
}

var DefaultOptions = &Options{
	TemplateDir : "template",
	ProxyFile: "proxy.list",
	UserAgent: "wankaizhang",
}

type Extractor struct {
	opt *Options
	tplMap map[string] *Template
	methods map[string] FieldMethod
	c *Client
}

func NewExtractor(opt *Options) (ex *Extractor, err error) {
	ex = &Extractor{
		opt: opt,
		tplMap: make(map[string] *Template),
		methods: make(map[string] FieldMethod),
		},
	}
	// 初始化Http客户端
	crawlOpts := &crawl.Options{
		UserAgent: opt.UserAgent,
		ProxyFileL: opt.ProxyFile,
		CookieDir: opt.CookieDir,
	}
	if ex.c, err = crawl.NewClient(crawlOpts); err != nil {
		return nil, err
	}
	// 加载所有模板
	loadTemplate := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		tpl, err := BuildTemplate(ex.opt.TemplateDir + "/" + info.Name())
		if err != nil {
			return err
		}
		ex.tplMap[tpl.Domain] = tpl
		return nil
	}
	if err = filepath.Walk(ex.opt.TemplateDir, loadTemplate); err != nil {
		return nil, err
	}

	return ex, nil
}

func (ex *Extractor) Extract(url string, doc []byte) (r Result, err error) {
	var ok bool
	var tpl *Template
	var dom *html.HtmlDocument

	site := GetSite(url)
	domain := GetDomain(site)
	if tpl, ok = ex.tplMap[site]; !ok {
		if tpl, ok = ex.tplMap[domain]; !ok {
			return r, errors.New("template not found");
		}
	}

	if dom, err = gokogiri.ParseHtml(doc); err != nil {
		return r, err
	}
	return ex.extractAll(url, dom, tpl), nil
}


func (ex *Extractor) download(url string, referer string) (js string, err error) {
	var res *http.Response
	var data []byte
	if res, err = ex.c.Get(url, referer); err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return "", err
	}
	if data, err = ioutil.ReadAll(res.Body); err != nil {
		return "", err
	}

	return string(data), nil
}

// text -> pattern -> (request, filter) -> output -> method ->
// pattern 将text打散成数组
// (request, filter) 也是最终得到数组
// output将数组组合成字符串
func (ex *Extractor) filter(url string, text string, field *fieldSection, supports map[string] string) string {
	var arr = make([]string, 0)
	var s string
	var ok bool

	if pattern := field.patternRegex; pattern != nil {
		arr = pattern.FindStringSubmatch(text)
	} else {
		arr = append(arr, text)
	}

	MapAddArrayCover(supports, arr)
	if request := field.requestCombine; request != nil {
		var req string
		req = request.Exec(supports)
		if len(req) > 0 {
			var js string
			var err error
			if js, err = ex.download(req, url); err != nil {
				return ""
			}
			if ex.opt.Verbose {
				fmt.Println(js)
			}
			if field.filterRegex != nil {
				arr = field.filterRegex.FindStringSubmatch(js)
				if len(arr) == 0 {
					supports = map[string]string{}
				} else {
					MapAddArrayCover(supports, arr)
				}
			} else {
				arr = []string{js}
				MapAddArrayCover(supports, arr)
			}
		}
	}

	var output Combine
	if output = field.outputCombine; output == nil {
		output = Combine{combineItem{replace: true, content:"0"} }
	}
	s = output.Exec(supports)

	var m FieldMethod
	if len(field.method) > 0 {
		if m, ok = ex.methods[field.method]; ok {
			return m(s)
		}
		s = ""
	}
	return s
}

// single，只提取一个还是全部
func (ex *Extractor) getLinks(src string, node xml.Node, field *fieldSection, supports map[string] string, single bool) (urls []string, anchors []string) {
	urls = make([]string, 0)
	anchors = make([]string, 0)

	if node == nil || field == nil {
		return
	}

	var nodes []xml.Node
	if node.Name() == field.typ {
		nodes = append(nodes, node)
	}
	tmp, _ := node.Search(".//"+field.typ)
	nodes = append(nodes, tmp...)

	for _, n := range(nodes) {
		if field.typ == "a" {
			text := n.Attr("href")
			text = ex.filter(src, text, field, supports)
			if text != "" {
				urls = append(urls, text)
				anchors = append(anchors, n.Content())
				if single {
					return urls, anchors
				}
			}
		} else if field.typ == "img" {
			text := n.Attr("src")
			text = ex.filter(src, text, field, supports)
			if text != "" {
				urls = append(urls, text)
				anchors = append(anchors, n.Attr("alt"))
				if single {
					return urls, anchors
				}
			}
		}
	}
	return urls, anchors
}

// (xpath, type, prop) -> text
func (ex *Extractor) getContent(url string, node xml.Node, field *fieldSection, supports map[string] string) string {
	var text string
	switch field.typ {
	case "html":
		text = node.InnerHtml()
	case "text":
		text = node.Content()
	case "attr":
		text = node.Attr(field.prop)
	default:
	}
	if text != "" {
		text = ex.filter(url, text, field, supports)
		if text != "" {
			return text
		}
	}
	return ex.filter(url, "", field, supports)
}

func (ex *Extractor) extractAll(url string, dom *html.HtmlDocument, tpl *Template) Result {
	r := Result {
		ItemMap: make(map[string] []Item),
		LinkMap: make(map[string] []Link),
	}

	for _, usec := range(tpl.urls) {
		arr := usec.patternRegex.FindStringSubmatch(url)
		if len(arr) == 0 {
			continue
		}
		// 提取url section中的support字段
		tmp := make(map[string] string)
		supports := make(map[string] string)
		supports["and"] = "&"
		MapAddArrayPanic(tmp, arr)

		for k, v := range(usec.extra) {
			cb := MakeCombine(v)
			value := cb.Exec(tmp)
			if value != "" {
				supports[k] = value
			}
		}
		// 提取support区域
		for _, sp := range(usec.supports) {
			// 定位support区域
			root := dom.Root();
			nodes, _ := dom.Search(sp.xpath);
			if len(nodes) > 0 {
				root = nodes[0].(*xml.ElementNode)
			}
			for _, field := range(sp.fields) {
				newsup := map[string]string{}
				for k, v := range(supports) {
					newsup[k] = v
				}
				if _, ok := supports[field.name]; ok {
					continue
				}
				if field.typ != "a" && field.typ != "img" {
					text := ex.getContent(url, root, field, newsup)
					supports[field.name] = text
				}
			}
		}
		// 提取link区
		for _,  lk := range(usec.links) {
			newsup := map[string]string{}
			for k, v := range(supports) {
				newsup[k] = v
			}
			root := dom.Root()
			ceils := make([]xml.Node, 0)
			if lk.xpath == "" {
				ceils = append(ceils, root)
			} else {
				ceils, _ = dom.Search(lk.xpath)
			}
			for _, node := range(ceils) {
				if lk.typ == "a" || lk.typ == "img" {
					u, a := ex.getLinks(url, node, lk, newsup, false)
					length := len(u)
					for i := 0; i < length; i++ {
						if _, ok := r.LinkMap[lk.name]; !ok {
							r.LinkMap[lk.name] = make([]Link, 0)
						}
						r.LinkMap[lk.name] = append(r.LinkMap[lk.name], Link{Url:u[i], Anchor:a[i]})
					}
				} else {
					u := ex.getContent(url, node, lk, newsup)
					a := ""
					if u != "" {
						if _, ok := r.LinkMap[lk.name]; ok {
							r.LinkMap[lk.name] = make([]Link, 0)
						}
						r.LinkMap[lk.name] = append(r.LinkMap[lk.name], Link{Url: u, Anchor:a})
					}
				}
			}
		}
		// 提取item区域
		for _, im := range(usec.items) {
			root := dom.Root()
			ceils := make([]xml.Node, 0)
			if im.xpath == "" {
				ceils = append(ceils, root)
			} else {
				ceils, _ = dom.Search(im.xpath)
			}
			for _, ceil := range(ceils) {
				var item Item = make(map[string] string)
				for _, field := range(im.fields) {
					newsup := map[string]string{}
					for k, v := range(supports) {
						newsup[k] = v
					}
					if _, ok := item[field.name]; ok {
						continue
					}
					var nodes []xml.Node
					var text string
					if field.xpath == "" {
						nodes = append(nodes, ceil)
					} else {
						nodes, _ = ceil.Search(field.xpath)
					}
					for _, n := range(nodes) {
						if field.typ != "a" && field.typ != "img" {
							text = ex.getContent(url, n, field, newsup)
							if text != "" {
								item[field.name] = text
								break
							}
						} else {
							urls, _ := ex.getLinks(url, n, field, newsup, true)
							if len(urls) > 0 {
								item[field.name] = urls[0]
								break
							}
						}
					}
				}
				if _, ok := r.ItemMap[im.name]; !ok {
					r.ItemMap[im.name] = make([]Item, 0)
				}
				r.ItemMap[im.name] = append(r.ItemMap[im.name], item)
			}
		}
		return r
	}
	return r
}
