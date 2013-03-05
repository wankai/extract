package extract

import (
	"fmt"
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/xml"
)

type Template struct {
	Domain string
	Name string
	urls []*urlSection
}

type urlSection struct {
	pattern string
	patternRegex *regexp.Regexp
	extra map[string] string
	supports []*itemSection
	links []*fieldSection
	items []*itemSection
}

type itemSection struct {
	name string
	xpath string
	fields []*fieldSection
}

type fieldSection struct {
	name string
	xpath string
	typ string
	prop string
	pattern string
	patternRegex *regexp.Regexp
	request string
	requestCombine Combine
	filter string
	filterRegex *regexp.Regexp
	output string
	outputCombine Combine
	method string
}

func BuildTemplate(file string) (tpl *Template, err error) {
	var f *os.File
	var data []byte
	var dom *xml.XmlDocument

	if f, err = os.Open(file); err != nil {
		return nil, err
	}
	if data, err = ioutil.ReadAll(f); err != nil {
		return nil, err
	}
	if dom, err = gokogiri.ParseXml(data); err != nil {
		return nil, err
	}
	if tpl, err = parseTemplate(dom); err != nil {
		return nil, err
	}
	return  tpl, nil
}

func parseTemplate(dom *xml.XmlDocument) (tpl *Template, err error) {
	tpl = &Template{urls: make([]*urlSection, 0)}

	root := dom.Root()
	if root == nil {
		return nil, nil
	}

	if root.Name() != "template" {
		return nil, errors.New("[template] tag is not 'template'")
	}
	domain := root.Attr("domain");
	if domain == "" {
		return nil, errors.New("[template] domain attribute not found")
	}
	tpl.Domain = domain

	name := root.Attr("name");
	if name == "" {
		return nil, errors.New("[template] name attribute not found")
	}
	tpl.Name = name

	child := root.FirstChild()
	for ; child != nil; child = child.NextSibling() {
		name := child.Name()
		if name == "text" || name == "comment" {
			continue
		}
		var url *urlSection
		if url, err = parseUrlPart(child); err != nil {
			return nil, err
		}
		tpl.urls = append(tpl.urls, url)
	}
	return tpl, nil
}

func parseUrlPart(root xml.Node) (url *urlSection, err error) {
	url = &urlSection{
		extra: make(map[string] string),
		supports: make([]*itemSection, 0),
		links: make([]*fieldSection, 0),
		items: make([]*itemSection, 0),
	}
	name := root.Name()
	if name != "url" {
		return nil, errors.New("[url] tag is not url")
	}

	for key, value := range(root.Attributes()) {
		if key == "pattern" {
			url.pattern = value.String()
			if len(url.pattern) > 0 {
				if url.patternRegex, err = regexp.Compile(url.pattern); err != nil {
					return nil, err
				}
			}
			continue
		}
		url.extra[key] = value.String()
	}

	child := root.FirstChild()
	for ; child != nil; child = child.NextSibling() {
		name := child.Name()
		if name == "text" || name == "comment" {
			continue
		}
		switch name {
		case "support":
			var support *itemSection
			if support, err = parseItem(child); err != nil {
				return nil, err
			}
			url.supports = append(url.supports, support)
		case "link":
			var link *fieldSection
			if link, err = parseField(child); err != nil {
				return nil, err
			}
			url.links = append(url.links, link)
		case "item":
			var item *itemSection
			if item, err = parseItem(child); err != nil {
				return nil, err
			}
			url.items = append(url.items, item)
		}
	}
	return url, nil
}

func parseItem(root xml.Node) (item *itemSection, err error) {
	item = &itemSection{
		fields: make([]*fieldSection, 0),
	}

	tag := root.Name()
	if tag != "item" && tag != "support" {
		return nil, errors.New("item section' tag is not item or support")
	}
	name := root.Attr("name")
	if tag == "item" && name == "" {
		return nil, errors.New("item section's name not found")
	}
	xpath := root.Attr("xpath")
	item.name = name
	item.xpath = xpath

	child := root.FirstChild()
	for ; child != nil; child = child.NextSibling() {
		tag := child.Name()
		if tag == "text" || tag == "comment" {
			continue
		}
		var field *fieldSection
		if field, err = parseField(child); err != nil {
			return nil, err
		}
		item.fields = append(item.fields, field)
	}
	return item, nil
}

func parseField(root xml.Node) (field *fieldSection, err error) {
	field = &fieldSection{}

	tag := root.Name()
	if tag != "field" && tag != "link" {
		return nil, errors.New("[field] tag is not 'field' or 'link'")
	}
	for key, value := range(root.Attributes()) {
		switch key {
		case "name":
			field.name = value.String()
		case "xpath":
			field.xpath = value.String()
		case "type":
			field.typ = value.String()
		case "prop":
			field.prop = value.String()
		case "pattern":
			field.pattern = value.String()
			if len(field.pattern) > 0 {
				if field.patternRegex, err = regexp.Compile(field.pattern); err != nil {
					return nil, err
				}
			}
		case "request":
			field.request = value.String()
			field.requestCombine = MakeCombine(field.request)
		case "filter":
			field.filter = value.String()
			if len(field.filter) > 0 {
				if field.filterRegex, err = regexp.Compile(field.filter); err != nil {
					return nil, err
				}
			}
		case "output":
			field.output = value.String()
			field.outputCombine = MakeCombine(field.output)
		case "method":
			field.method = value.String()
		default:
			return nil, errors.New(fmt.Sprintf("[field] unkown attribute '%s'", key))
		}
	}
	return field, nil
}

func (tpl *Template) String() string {
	s := fmt.Sprintf("<template domain=\"%s\" name=\"%s\">\n", tpl.Domain, tpl.Name)
	for _, url := range(tpl.urls) {
		s += fmt.Sprintf("  <url pattern=\"%s\"", url.pattern)
		for key, value := range(url.extra) {
			s += fmt.Sprintf(" %s=\"%s\"", key, value)
		}
		s += ">\n"

		for _, support := range(url.supports) {
			s += fmt.Sprintf("    <support xpath=\"%s\">\n", support.xpath)
			for _, field := range(support.fields) {
				s += fmt.Sprintf("      <field name=\"%s\" xpath=\"%s\" type=\"%s\" prop=\"%s\" pattern=\"%s\" request=\"%s\" filter=\"%s\"output=\"%s\" method=\"%s\" />\n", field.name, field.xpath, field.typ, field.prop, field.pattern, field.request, field.filter, field.output, field.method)
			}
			s += "    </support>\n"
		}

		for _, link := range(url.links) {
			s += fmt.Sprintf("    <link name=\"%s\" xpath=\"%s\" type=\"%s\" prop=\"%s\" pattern=\"%s\" request=\"%s\" filter=\"%s\"output=\"%s\" method=\"%s\" />\n", link.name, link.xpath, link.typ, link.prop, link.pattern, link.request, link.filter, link.output, link.method)
		}

		for _, item := range(url.items) {
			s += fmt.Sprintf("    <item name=\"%s\" xpath=\"%s\">\n", item.name, item.xpath)
			for _, field := range(item.fields) {
				s += fmt.Sprintf("      <field name=\"%s\" xpath=\"%s\" type=\"%s\" prop=\"%s\" pattern=\"%s\" request=\"%s\" filter=\"\" output=\"%s\" method=\"%s\" />\n", field.name, field.xpath, field.typ, field.prop, field.pattern, field.request, field.filter, field.output, field.method)
			}
			s += "    </item>\n"
		}
		s += "  </url>\n"
	}
	s += "</template>\n"

	return s
}
