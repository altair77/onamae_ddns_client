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
	// ConfigFile is config file path
	ConfigFile string = "./config.yml"
	// GlobalIpUrl is URL to get global IP address
	GlobalIpUrl string = "http://inet-ip.info/ip"
	// OnamaeUrl is URL to update ip onamae.com
	OnamaeUrl string = "ddnsclient.onamae.com:65010"
	// ResponseOk is text to compare the communication success string
	ResponseOk string = "000 COMMAND SUCCESSFUL\n.\n"
	// ResponseSize is length of ResponseOk
	ResponseSize int = 32
)

// Config is format for ConfigFile content
type Config struct {
	// Auth is userid:password encoded to Base64
	Auth string `yaml:"auth"`
	// Domains are target settings
	Domains []struct {
		// Name is domain name
		Name string `yaml:"name"`
		// Hosts is host names
		Hosts []struct {
			Name string `yaml:"name"`
		}
	}
}

// Client used to connect by TSL
type Client struct {
	// conn is TLS connection
	conn *tls.Conn
}

// Open connection to onamae.com
// Return: error
func (c *Client) Open() error {
	conn, err := tls.Dial("tcp", OnamaeUrl, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	return c.verifyResponse()
}

// Close connection
// Return: error
func (c *Client) Close() error {
	return c.conn.Close()
}

// Send data
// msg: data to send
// Return: error
func (c *Client) Send(msg string) error {
	size, err := c.conn.Write([]byte(msg))
	if err != nil {
		return err
	}
	if size != len(msg) {
		return errors.New("bad written size")
	}
	return err
}

// Login to onamae.com
// username: userid
// passwd: password
// Return: error
func (c *Client) Login(username string, passwd string) error {
	msg := fmt.Sprintf("LOGIN\nUSERID:%s\nPASSWORD:%s\n.\n", username, passwd)
	if err := c.Send(msg); err != nil {
		return err
	}
	return c.verifyResponse()
}

// Logout by onamae.com
// Return: error
func (c *Client) Logout() error {
	msg := fmt.Sprintf("LOGOUT\n.\n")
	if err := c.Send(msg); err != nil {
		return err
	}
	return c.verifyResponse()
}

// ModIP change IP
// host: hostname
// domain: domain name
// ip: ip address
// Return: error
func (c *Client) ModIP(host string, domain string, ip string) error {
	msg := "MODIP\n"
	if host != "" {
		msg += fmt.Sprintf("HOSTNAME:%s\n", host)
	}
	msg += fmt.Sprintf("DOMNAME:%s\nIPV4:%s\n.\n", domain, ip)
	if err := c.Send(msg); err != nil {
		return err
	}
	return c.verifyResponse()
}

// verifyResponse wait and verify response
// Return: error
func (c *Client) verifyResponse() error {
	buf := make([]byte, ResponseSize)
	size, err := c.conn.Read(buf)
	if err != nil {
		return err
	}
	if string(buf[:size]) != ResponseOk {
		return errors.New("bad response")
	}
	return err
}

// readConfig read config file
// config (out): config data
// Return: error
func readConfig(config *Config) error {
	buf, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(buf, config)
}

// getGlobalIP get global IP address
// Return: IP address
// Return: error
func getGlobalIP() (string, error) {
	res, err := http.Get(GlobalIpUrl)
	if err != nil {
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	return string(bodyBytes), err
}

// login onamae.com used by client and config
// client: connection client
// config: configuration
// Return: error
func login(client *Client, config *Config) error {
	authDec, err := base64.StdEncoding.DecodeString(config.Auth)
	if err != nil {
		return err
	}
	authArr := strings.SplitN(string(authDec), ":", 2)
	client.Login(authArr[0], authArr[1])
	return err
}

// update DNS IP address to global IP address
// client: connection client
// config: configuration
// globalIP: global IP address
// Return: error
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
	if err := readConfig(config); err != nil {
		log.Fatalf("Faild to read config\n%s", err)
	}

	client := &Client{}
	currentIP := ""
	handler := func() {
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
	}

	c := cron.New()
	if _, err := c.AddFunc("@every 10m", handler); err != nil {
		log.Fatalf("Faild to create cron\n%s", err)
	}
	c.Start()

	for {
		time.Sleep(24 * 60 * 60 * 1000)
	}
}
