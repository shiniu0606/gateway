package main 

import (
	"path/filepath"
	"io/ioutil"
	"encoding/json"
)

//应用配置
var AppConfigs AppConfig

type AppConfig struct {
	DefaultPort 	string `json:"DefaultPort"` 
	WebScoketPort	string `json:"WebsocketPort"`    
	Secret     		string `json:"Secret"`    
	InfoLogPath       string `json:"InfoLogPath"`       //mysql连接字符串
	ErrorLogPath       string `json:"ErrorLogPath"`       //redis连接字符串
}

func init() {
	//初始化配置文件
	configFilePath, _ := filepath.Abs("./config/config.json")
	err := InitConfigFile(configFilePath, &AppConfigs)
	if err != nil {
		panic(err)
	}

}

//InitConfigFile 初始化配置文件信息
func InitConfigFile(configFilePath string, out interface{}) error {
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	err = FromByteJSON(content, out)
	if err != nil {
		return err
	}
	return nil
}

//FromByteJSON 返回序列化对象
func FromByteJSON(data []byte, obj interface{}) error {
	err := json.Unmarshal(data, obj)
	if err != nil {
		return err
	}
	return nil
}