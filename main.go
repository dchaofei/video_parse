package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const defaultAddr = "0.0.0.0:8888"
var addr = defaultAddr

func init() {
	args := os.Args
	if len(args) > 1 {
		addr = args[1]
	}
}

func main() {
	http.HandleFunc("/", resolveUrl)
	log.Printf("已监听 %s, 使用%s?url=", addr, addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func resolveUrl(w http.ResponseWriter, r *http.Request) {
	fullUrl := r.URL.String()
	log.Println("request: " + fullUrl)

	paramUrl := r.URL.Query().Get("url")

	if paramUrl == "" {
		fmt.Fprint(w, "缺少 url 参数")
		return
	}

	resolve := &ResolveVideo{Url: paramUrl}
	videoUrl, err := resolve.GetVideoUrl()
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	output, err := ioutil.ReadFile("output.html")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	fmt.Fprintf(w, string(output), videoUrl, videoUrl)
}

type ResolveVideo struct {
	Url, body string
}

func (r *ResolveVideo) GetVideoUrl() (string, error) {
	pcVideoUrl, err := r.getPcVideoUrl()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(pcVideoUrl, "http") {
		return "", errors.New("url 不合法")
	}

	log.Println("pcVideoUrl: " + pcVideoUrl)

	md5, err := r.getMd5()
	if err != nil {
		return "", err
	}
	log.Println("md5: " + md5)

	response, err := http.PostForm("http://api.youyitv.com/lekan/api.php", url.Values{
		"id":  {pcVideoUrl},
		"md5": {md5},
	})
	if err != nil {
		return "", err
	}

	if response.StatusCode != 200 {
		return "", errors.New("获取视频地址失败: " + string(response.StatusCode))
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}

	type jsonData struct {
		Success int
		Url     string
		Msg     string
	}

	log.Println(string(body))

	videoModel := jsonData{}

	err = json.Unmarshal(body, &videoModel)

	if err != nil {
		return "", err
	}

	if videoModel.Success != 1 {
		return "", errors.New(videoModel.Msg)
	}

	return videoModel.Url, nil
}

func (r *ResolveVideo) getPcVideoUrl() (string, error) {
	body, err := r.getBody()
	if err != nil {
		return "", err
	}

	re, _ := regexp.Compile("{\"id\": \"(.*)\",\"type")
	all := re.FindStringSubmatch(string(body))

	if all == nil {
		return "", errors.New("没有获取到合法的视频url")
	}

	return all[1], nil
}

func (r *ResolveVideo) getMd5() (string, error) {
	body, err := r.getBody()
	if err != nil {
		return "", err
	}

	re, _ := regexp.Compile("eval.*")
	all := re.FindAllString(string(body), -1)

	if !strings.Contains(all[1], "\\x") {
		return "", errors.New("获取md5失败")
	}

	//fmt.Println(all[1])
	md5Byte := strings.Trim(all[1], "eval( \") ;")

	md5ByteArray := strings.Split(md5Byte, "\\x")

	var md5String string
	for _, v := range (md5ByteArray[18:])[0 : len(md5ByteArray[18:])-3] {
		_md5String, _ := hex.DecodeString(v)
		md5String += string(_md5String)
	}

	return md5String, nil
}

func (r *ResolveVideo) getBody() (string, error) {
	if r.body != "" {
		return r.body, nil
	}

	step1Url := "http://api.youyitv.com/lekan/oko.php?url=" + r.Url
	response, err := http.Get(step1Url)
	if err != nil {
		return "", err
	}

	if response.StatusCode != 200 {
		return "", errors.New("getBody:获取url错误 " + string(response.StatusCode))
	}
	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return "", err
	}
	r.body = string(body)
	return r.body, nil
}
