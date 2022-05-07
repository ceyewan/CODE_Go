package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var mu sync.Mutex
var wg sync.WaitGroup

type CaiyunRequest struct {
	TransType string `json:"trans_type"`
	Source    string `json:"source"`
	UserId    string `json:"user_id"`
}

type CaiyunResponse struct {
	Rc   int `json:"rc"`
	Wiki struct {
		KnownInLaguages int `json:"known_in_laguages"`
		Description     struct {
			Source string      `json:"source"`
			Target interface{} `json:"target"`
		} `json:"description"`
		ID   string `json:"id"`
		Item struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"item"`
		ImageURL  string `json:"image_url"`
		IsSubject string `json:"is_subject"`
		Sitelink  string `json:"sitelink"`
	} `json:"wiki"`
	Dictionary struct {
		Prons struct {
			EnUs string `json:"en-us"`
			En   string `json:"en"`
		} `json:"prons"`
		Explanations []string      `json:"explanations"`
		Synonym      []string      `json:"synonym"`
		Antonym      []string      `json:"antonym"`
		WqxExample   [][]string    `json:"wqx_example"`
		Entry        string        `json:"entry"`
		Type         string        `json:"type"`
		Related      []interface{} `json:"related"`
		Source       string        `json:"source"`
	} `json:"dictionary"`
}

type YoudaoResponse struct {
	ReturnPhrase []string `json:"returnPhrase"`
	Query        string   `json:"query"`
	ErrorCode    string   `json:"errorCode"`
	L            string   `json:"l"`
	TSpeakURL    string   `json:"tSpeakUrl"`
	Web          []struct {
		Value []string `json:"value"`
		Key   string   `json:"key"`
	} `json:"web"`
	RequestID   string   `json:"requestId"`
	Translation []string `json:"translation"`
	Dict        struct {
		URL string `json:"url"`
	} `json:"dict"`
	Webdict struct {
		URL string `json:"url"`
	} `json:"webdict"`
	Basic struct {
		ExamType   []string `json:"exam_type"`
		UsPhonetic string   `json:"us-phonetic"`
		Phonetic   string   `json:"phonetic"`
		UkPhonetic string   `json:"uk-phonetic"`
		Wfs        []struct {
			Wf struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"wf"`
		} `json:"wfs"`
		UkSpeech string   `json:"uk-speech"`
		Explains []string `json:"explains"`
		UsSpeech string   `json:"us-speech"`
	} `json:"basic"`
	IsWord   bool   `json:"isWord"`
	SpeakURL string `json:"speakUrl"`
}

func queryCaiyun(word string) {
	client := &http.Client{} // 创建 http client
	request := CaiyunRequest{TransType: "en2zh", Source: word, UserId: "624da0f581049b0050168f05"}
	buf, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}
	var data = bytes.NewBuffer(buf)
	// var data = strings.NewReader(`{"trans_type":"en2zh","source":"hello","user_id":"624da0f581049b0050168f05"}`)
	// 构建 http 请求，是 post 类型
	req, err := http.NewRequest("POST", "https://api.interpreter.caiyunai.com/v1/dict", data)
	if err != nil {
		log.Fatal(err)
	}
	for {
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("Origin", "https://fanyi.caiyunapp.com")
		req.Header.Set("Referer", "https://fanyi.caiyunapp.com/")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36")
		req.Header.Set("X-Authorization", "token:qgemv4jr1y38jyq6vhvi")
		req.Header.Set("app-name", "xy")
		req.Header.Set("os-type", "web")
		req.Header.Set("sec-ch-ua", `" Not A;Brand";v="99", "Chromium";v="101", "Google Chrome";v="101"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		break
	}
	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// 函数结束的时候关闭 body 流
	defer resp.Body.Close()
	// 读取响应
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var dictResponse CaiyunResponse
	err = json.Unmarshal(bodyText, &dictResponse)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("bad StatusCode:", resp.StatusCode, "body", string(bodyText))
	}
	// fmt.Printf("%#v\n", dictResponse)
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("By Caiyun:")
	fmt.Println(word, "UK:", dictResponse.Dictionary.Prons.En, "US:", dictResponse.Dictionary.Prons.EnUs)
	for _, item := range dictResponse.Dictionary.Explanations {
		fmt.Println(item)
	}
}

func queryYoudao(word string) {
	client := &http.Client{}
	var data = strings.NewReader(fmt.Sprintf(`q=%s&from=en&to=zh-CHS`, word))
	req, err := http.NewRequest("POST", "https://aidemo.youdao.com/trans", data)
	if err != nil {
		log.Fatal(err)
	}
	for {
		req.Header.Set("authority", "aidemo.youdao.com")
		req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8")
		req.Header.Set("content-type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("origin", "https://ai.youdao.com")
		req.Header.Set("referer", "https://ai.youdao.com/")
		req.Header.Set("sec-ch-ua", `" Not A;Brand";v="99", "Chromium";v="101", "Google Chrome";v="101"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-site")
		req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36")
		break
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("bad StatusCode:", resp.StatusCode, "body", string(bodyText))
	}
	var dictResponse YoudaoResponse
	err = json.Unmarshal(bodyText, &dictResponse)
	if err != nil {
		log.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	fmt.Println("By Youdao:")
	fmt.Println(word, dictResponse.Translation[0])
	for _, item := range dictResponse.Basic.Explains {
		fmt.Println(item)
	}
}
func main() {
	if len(os.Args) > 3 || len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, `usage: simpleDict WORD
		example: simpleDict hello [dict]`)
		os.Exit(1)
	}
	word := os.Args[1]
	if len(os.Args) == 2 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			queryCaiyun(word)
		}()
		go func() {
			defer wg.Done()
			queryYoudao(word)
		}()
		wg.Wait()
	} else {
		dict := os.Args[2]
		switch dict {
		case "caiyun":
			queryCaiyun(word)
		case "youdao":
			queryYoudao(word)
		default:
			fmt.Fprintf(os.Stderr, `select dict from caiyun or youdao`)
			os.Exit(1)
		}
	}
}
