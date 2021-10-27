package main

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func proxy(c *gin.Context) {
	// 判断header
	// if c.GetHeader("direct") != "lab" {
	// 	c.String(404, "direct header none")
	// 	return
	// }

	// var proxyUrl = new(url.URL)
	// proxyUrl.Scheme = "https"
	// proxyUrl.Host = "localhost"
	// proxyUrl.RawQuery = url.QueryEscape("proxy=true")
	//u.Path = "base" // 这边若是赋值了，做转发的时候，会带上path前缀，例： /hello -> /base/hello

	// var query url.Values
	// query.Add("token", "VjouhpQHa6wgWvtkPQeDZbQd")
	// u.RawQuery = query.Encode()

	urls, err := url.Parse("https://localhost")
	if err != nil {
		c.String(500, err.Error())
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(urls)
	//proxy := httputil.ReverseProxy{}
	//proxy.Director = func(req *http.Request) {
	//	fmt.Println(req.URL.String())
	//	req.URL.Scheme = "http"
	//	req.URL.Host = "172.16.60.161"
	//	rawQ := req.URL.Query()
	//	rawQ.Add("token", "VjouhpQHa6wgWvtkPQeDZbQd")
	//	req.URL.RawQuery = rawQ.Encode()
	//}

	// proxy.ErrorHandler // 可以添加错误回调
	// proxy.Transport // 若有需要可以自定义 http.Transport
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Server")
		resp.Header.Add("XP", "GO")
		return nil
	}
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

// 也就是做简单的转发操作
func customProxy(c *gin.Context) {
	// if c.GetHeader("direct") != "lab" {
	// 	return
	// }

	err := setTokenToUrl(c.Request.URL)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("填写的地址有误: %s", err.Error()))
		c.Abort()
		return
	}

	req, err := http.NewRequestWithContext(c, c.Request.Method, c.Request.URL.String(), c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		c.Abort()
		return
	}
	defer req.Body.Close()
	req.Header = c.Request.Header

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		c.Abort()
		return
	}
	// header 也带过来
	for k := range resp.Header {
		for j := range resp.Header[k] {
			c.Header(k, resp.Header[k][j])
		}
	}
	extraHeaders := make(map[string]string)
	extraHeaders["direct"] = "lab"
	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, extraHeaders)
	c.Abort()
}

func setTokenToUrl(rawUrl *url.URL) error {
	// 这边是从设置里拿代理值
	//equipment, err := proxy.GetEquipment()
	//if err != nil {
	//	return err
	//}
	proxyUrl := "https://localhost"
	token := "VjouhpQHa6wgWvtkPQeDZbQd"
	u, err := url.Parse(proxyUrl)
	if err != nil {
		return err
	}

	rawUrl.Scheme = u.Scheme
	rawUrl.Host = u.Host
	ruq := rawUrl.Query()
	ruq.Add("token", token)
	rawUrl.RawQuery = ruq.Encode()
	return nil
}

func mutilHAProxy(c *gin.Context) {
	one, err := url.Parse("https://localhost")
	if err != nil {
		c.String(500, err.Error())
		return
	}
	urls := []*url.URL{
		one,
		{
			Scheme: "https",
			Host:   "localhost",
		},
	}

	num := rand.Int() % len(urls)
	director := func(req *http.Request) {
		target := urls[num]
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
	}

	proxy := httputil.ReverseProxy{Director: director}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Server")
		resp.Header.Add("urlsId", fmt.Sprintf("%d", num))
		return nil
	}
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// r.Any("/k8s/*action", proxy)
	// r.Any("/k8s/*action", customProxy)
	r.Any("/k8s/*action", mutilHAProxy)
	r.Run("0.0.0.0:8888") // listen and serve on  (for windows "localhost:8080")
}
