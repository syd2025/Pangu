package models

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type PathConfig struct {
	Cert         string
	Key          string
	Avatars      string
	CourseImages string `yaml:"courseimages"`
	Lectures     string
	Html         string
	Home         string
}

func (c *PathConfig) CertPath() string {
	return filepath.Join(c.Home, c.Cert)
}

func (c *PathConfig) KeyPath() string {
	return filepath.Join(c.Home, c.Key)
}

func (c *PathConfig) AvatarsPath() string {
	return filepath.Join(c.Home, c.Avatars)
}

func (c *PathConfig) CourseImagesPath() string {
	return filepath.Join(c.Home, c.CourseImages)
}

func (c *PathConfig) LecturesPath() string {
	return filepath.Join(c.Home, c.Lectures)
}

func (c *PathConfig) HtmlPath() string {
	return filepath.Join(c.Home, c.Html)
}

type DatabaseConfig struct {
	Host         string
	Port         int
	Dbname       string
	User         string
	Passwd       string
	MaxOpenConns int    `yaml:"maxopenconns"`
	MaxIdleConns int    `yaml:"maxidleconns"`
	MaxIdleTime  string `yaml:"maxidletime"`
}

type LimiterConfig struct {
	Rps     int
	Burst   int
	Enabled bool
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Passwd   string
	Sender   string
}

type Config struct {
	Server    string
	Port      int
	VideoPort int `yaml:"videoport"`
	Env       string
	Path      PathConfig     `yaml:"path"`
	Database  DatabaseConfig `yaml:"database"`
	Limiter   LimiterConfig  `yaml:"limiter"`
	Smtp      SMTPConfig     `yaml:"smtp"`
}

func (cfg *DatabaseConfig) Dsn() string {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Host,
		cfg.User,
		cfg.Passwd,
		cfg.Dbname,
		cfg.Port,
	)
	return dsn
}

func (c *DatabaseConfig) MaxIdleDuration() time.Duration {
	duration, err := time.ParseDuration(c.MaxIdleTime)
	if err != nil {
		duration, _ = time.ParseDuration("15m")
	}
	return duration
}

func NewConfig(path, home string) (*Config, error) {
	stats, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes := make([]byte, stats.Size())
	buf := bufio.NewReader(f)
	_, err = buf.Read(bytes)
	if err != nil {
		return nil, err
	}

	cfg := Config{}
	err = yaml.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	} else {
		cfg.Path.Home = home
		return &cfg, nil
	}
}
