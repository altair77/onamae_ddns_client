package main

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gopkg.in/yaml.v2"
)

const (
	CONFIG_FILE   string = "./config.yml"
	GLOBAL_IP_URL string = "http://inet-ip.info/ip"
	ONAMAE_URL    string = "ddnsclient.onamae.com:65010"
	RESPONSE_OK   string = "000 COMMAND SUCCESSFUL\n.\n"
	RESPONSE_SIZE int    = 32
)

type Config struct {
	Auth    string `yaml:"auth"`
	Domains []struct {
		Name  string `yaml:"name"`
		Hosts []struct {
			Name string `yaml:"name"`
		}
	}
}

type Client struct {
	conn *tls.Conn
}

func (self *Client) Open() error {
	conn, err := tls.Dial("tcp", ONAMAE_URL, nil)
	if err != nil {
		return err
	}
	self.conn = conn
	return self.verifyResponse()
}

func (self *Client) Close() error {
	return self.conn.Close()
}

func (self *Client) Send(msg string) error {
	size, err := self.conn.Write([]byte(msg))
	if err != nil {
		return err
	}
	if size != len(msg) {
		return errors.New("Bad written size")
	}
	return err
}

func (self *Client) Login(username string, passwd string) error {
	msg := fmt.Sprintf("LOGIN\nUSERID:%s\nPASSWORD:%s\n.\n", username, passwd)
	if err := self.Send(msg); err != nil {
		return err
	}
	return self.verifyResponse()
}

func (self *Client) Logout() error {
	msg := fmt.Sprintf("LOGOUT\n.\n")
	if err := self.Send(msg); err != nil {
		return err
	}
	return self.verifyResponse()
}

func (self *Client) ModIP(host string, domain string, ip string) error {
	msg := fmt.Sprintf("MODIP\nHOSTNAME:%s\nDOMNAME:%s\nIPV4:%s\n.\n", host, domain, ip)
	if err := self.Send(msg); err != nil {
		return err
	}
	return self.verifyResponse()
}

func (self *Client) verifyResponse() error {
	buf := make([]byte, RESPONSE_SIZE)
	size, err := self.conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:size]) != RESPONSE_OK {
		return errors.New("Bad response")
	}
	return err
}

func readConfig(config *Config) error {
	buf, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(buf, config)
}

func getGlobalIP() (string, error) {
	res, err := http.Get(GLOBAL_IP_URL)
	if err != nil {
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	return string(bodyBytes), err
}

func login(client *Client, config *Config) error {
	authDec, err := base64.StdEncoding.DecodeString(config.Auth)
	if err != nil {
		return err
	}
	authArr := strings.SplitN(string(authDec), ":", 2)
	client.Login(authArr[0], authArr[1])
	return err
}

func update(client *Client, config *Config, globalIP string) error {
	if err := client.Open(); err != nil {
		return err
	}
	defer client.Close()
	if err := login(client, config); err != nil {
		return err
	}
	for _, domain := range config.Domains {
		for _, host := range domain.Hosts {
			if err := client.ModIP(host.Name, domain.Name, globalIP); err != nil {
				return err
			}
		}
	}
	return client.Logout()
}

func main() {
	config := &Config{}
	readConfig(config)

	client := &Client{}

	currentIP := ""

	c := cron.New()
	c.AddFunc("@every 10m", func() {
		globalIP, err := getGlobalIP()
		if err != nil {
			log.Fatalf("Faild to get global IP\n%s", err)
		}
		if currentIP != globalIP {
			log.Printf("Updated IP: %s\n", globalIP)
			if err := update(client, config, globalIP); err != nil {
				log.Fatalf("Faild to update IP\n%s", err)
			}
		}
		currentIP = globalIP
	})
	c.Start()

	for {
		time.Sleep(24 * 60 * 60 * 1000)
	}
}
