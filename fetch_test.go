package extract

import (
	"io/ioutil"
	"net/http"
	"fmt"
	"testing"
)

func TestFetch1(t *testing.T) {
	url := "http://count.tbcdn.cn/counter3?inc=ICVT_7_16442283488&sign=87316beaea7233a0629d0bc70106fced0a26c&keys=DFX_200_1_1644228348    8,ICVT_7_16442283488,ICCP_1_16442283488,SCCP_2_36658574&callback=DT.mods.SKU.CounterCenter.saveCounts"
	referer := "http://item.taobao.com/item.htm?id=16442283488"

	var c *Client
	var res *http.Response
	var data []byte
	var err error

	c = &Client{}
	if err = c.Init(); err != nil {
		t.Fatal(err)
	}
	if res, err = c.GetReferer(url, referer); err != nil {
		t.Fatal(err)
	}
	if data, err = ioutil.ReadAll(res.Body); err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	fmt.Println(string(data))
}
