package sms

import (
	"bytes"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type App struct {
	AppId  string
	AppKey string
	DesKey string
}


/**
 * 发送行业短信(需要在运营后台-短信服务管理 中进行配置)
 * @param app appId(应用ID)和appKey ,
 * 登录商户后台 : https://mch.ipaynow.cn ->商户中心->应用信息可以新增应用或查看appKey
 * @param mobile 发送手机号
 * @param content 发送内容
 * @param mhtOrderNo 商户订单号,可为空(自动生成)。商户订单号和状态报告通知中的相关字段对应
 * @param notifyUrl 后台通知地址
 */
func Send_hy(app *App, mobile string, content string, mhtOrderNo string, notifyUrl string) string {
	return send(app,"S01",mobile,content,mhtOrderNo,notifyUrl)
}
/**
 * 发送营销短信(需要在运营后台-短信服务管理 中进行配置)
 * @param app appId(应用ID)和appKey ,
 * 登录商户后台 : https://mch.ipaynow.cn ->商户中心->应用信息可以新增应用或查看appKey
 * @param mobile 发送手机号
 * @param content 发送内容
 * @param mhtOrderNo 商户订单号,可为空(自动生成)。商户订单号和状态报告通知中的相关字段对应
 * @param notifyUrl 后台通知地址
 */
func Send_yx(app *App, mobile string, content string, mhtOrderNo string, notifyUrl string) string {
	return send(app,"YX_01",mobile,content,mhtOrderNo,notifyUrl)
}

func send(app *App,types string,mobile string, content string, mhtOrderNo string, notifyUrl string) string {

	var postMap = make(map[string]string)

	postMap["funcode"] = types
	postMap["appId"] = app.AppId
	if mhtOrderNo != "" {
		postMap["mhtOrderNo"] = mhtOrderNo
	} else {
		postMap["mhtOrderNo"] = getRandomString(13)
	}
	postMap["mobile"] = mobile
	postMap["content"] = urlEncode(content)
	postMap["notifyUrl"] = notifyUrl

	var keys []string
	for k := range postMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var postFormLinkReport = ""
	for _, k := range keys {
		postFormLinkReport += k + "=" + postMap[k] + "&"
	}
	postFormLinkReport = postFormLinkReport[0 : len(postFormLinkReport)-1]

	var message1 = "appId=" + app.AppId
	b64 := base64.StdEncoding.EncodeToString([]byte(message1))
	message1 = string(b64)
	var des, err = TripleEcbDesEncrypt([]byte(postFormLinkReport), []byte(app.DesKey))
	if err != nil {
		fmt.Println(des)
		fmt.Println(err)
	}

	var message2 = base64.StdEncoding.EncodeToString([]byte(des))
	var tmp = fmt.Sprintf("%x", md5.Sum([]byte(postFormLinkReport+"&"+app.AppKey)))
	var message3 = base64.StdEncoding.EncodeToString([]byte(tmp))
	var message = message1 + "|" + message2 + "|" + message3 + ""

	u := url.Values{}
	u.Set("message", message)
	var result = post("https://sms.ipaynow.cn", "funcode="+types+"&"+u.Encode())

	//	decodeBytes, err := base64.StdEncoding.DecodeString("bWVzc2FnZVVSTOino+eggeWksei0pQ==")
	//	fmt.Println(string(decodeBytes))
	fmt.Println(result)

	return result
}

func urlEncode(content string) string {
	l, e := url.Parse("?" + content)
	if e != nil {
		fmt.Println(l, e)
	}
	return l.Query().Encode()[0 : len(l.Query().Encode())-1]
}

func post(url string, postcontent string) string {
	resp, err := http.Post(url,
		"application/x-www-form-urlencoded",
		strings.NewReader(postcontent))
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	return string(body)
}

func getRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

//ECB PKCS5Padding
func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func NoPadding(origData []byte) []byte {
	length := len(origData)

	if length%8 != 0 {
		var len = length - length%8 + 8
		var needData = make([]byte, len)
		for i := 0; i < len; i++ {
			needData[i] = 0x00
		}
		copy(needData, origData)
		return needData
	} else {
		return origData
	}
}

//ECB PKCS5Unpadding
func PKCS5Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//Des加密
func encrypt(origData, key []byte) ([]byte, error) {
	if len(origData) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(origData)%bs != 0 {
		return nil, errors.New("wrong padding")
	}
	out := make([]byte, len(origData))
	dst := out
	for len(origData) > 0 {
		block.Encrypt(dst, origData[:bs])
		origData = origData[bs:]
		dst = dst[bs:]
	}
	return out, nil
}

//Des解密
func decrypt(crypted, key []byte) ([]byte, error) {
	if len(crypted) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(crypted))
	dst := out
	bs := block.BlockSize()
	if len(crypted)%bs != 0 {
		return nil, errors.New("wrong crypted size")
	}

	for len(crypted) > 0 {
		block.Decrypt(dst, crypted[:bs])
		crypted = crypted[bs:]
		dst = dst[bs:]
	}

	return out, nil
}

//[golang ECB 3DES Encrypt]
func TripleEcbDesEncrypt(origData, key []byte) ([]byte, error) {
	tkey := make([]byte, 24, 24)
	copy(tkey, key)
	k1 := tkey[:8]
	k2 := tkey[8:16]
	k3 := tkey[16:]

	//	block, err := des.NewCipher(k1)
	//	if err != nil {
	//		return nil, err
	//	}
	//	bs := block.BlockSize()
	origData = NoPadding(origData)

	buf1, err := encrypt(origData, k1)
	if err != nil {
		return nil, err
	}
	buf2, err := decrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := encrypt(buf2, k3)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//[golang ECB 3DES Decrypt]
func TripleEcbDesDecrypt(crypted, key []byte) ([]byte, error) {
	tkey := make([]byte, 24, 24)
	copy(tkey, key)
	k1 := tkey[:8]
	k2 := tkey[8:16]
	k3 := tkey[16:]
	buf1, err := decrypt(crypted, k3)
	if err != nil {
		return nil, err
	}
	buf2, err := encrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := decrypt(buf2, k1)
	if err != nil {
		return nil, err
	}
	out = PKCS5Unpadding(out)
	return out, nil
}
