package extract

import (
	"fmt"
)

type Item map[string] string

type Link struct {
	Url string
	Anchor string
}

type Result struct {
	ItemMap map[string] []Item
	LinkMap map[string] []Link
}

func (r *Result) Println() {
	for name, links := range(r.LinkMap) {
		for _, link := range(links) {
			fmt.Println(name, link.Url, link.Anchor)
		}
	}
	for name, items := range(r.ItemMap) {
		fmt.Println(name)
		for _, item := range(items) {
			for k, v := range(item) {
				fmt.Println(k, ":", v)
			}
		}
	}
}
