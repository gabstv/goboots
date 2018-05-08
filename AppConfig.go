package goboots

import (
	"os"
	"regexp"
)

type DatabaseConfig struct {
	Name       string `yaml:"Name"`
	Connection string `yaml:"Connection"`
	Host       string `yaml:"Host"`
	Database   string `yaml:"Database"`
	User       string `yaml:"User"`
	Password   string `yaml:"Password"`
}

type AppConfig struct {
	Name            string                    `yaml:"Name"`
	DomainName      string                    `yaml:"DomainName"`
	GlobalPageTitle string                    `json:",omitempty"`
	Version         string                    `yaml:"Version"`
	HostAddr        string                    `yaml:"HostAddr"`
	HostAddrTLS     string                    `yaml:"HostAddrTLS"`
	MongoDbs        string                    `yaml:"MongoDbs"`
	Database        string                    `yaml:"Database"`
	Databases       map[string]DatabaseConfig `yaml:"Databases"`
	SessionDb       interface{}               `yaml:"SessionDb"`
	Salt            string                    `yaml:"Salt"`
	LocalePath      string                    `yaml:"LocalePath"`
	DefaultLanguage string                    `yaml:"DefaultLanguage"`
	Data            map[string]string         `yaml:"Data"`

	// TLS
	TLSCertificatePath   string   `yaml:"TLSCertificatePath"`
	TLSKeyPath           string   `yaml:"TLSKeyPath"`
	TLSRedirect          bool     `yaml:"TLSRedirect"`
	TLSRedirectPort      string   `yaml:"TLSRedirectPort"`
	TLSAutocert          bool     `yaml:"TLSAutocert"`
	TLSAutocertWhitelist []string `yaml:"TLSAutocertWhitelist"`
	RawTLSKey            string   `yaml:"RawTLSKey"`
	RawTLSCert           string   `yaml:"RawTLSCert"`

	// Paths
	RoutesConfigPath string   `yaml:"RoutesConfigPath"`
	CachePath        string   `yaml:"CachePath"`
	ViewsFolderPath  string   `yaml:"ViewsFolderPath"`
	ViewsExtensions  []string `yaml:"ViewsExtensions"` // .html, .tpl
	PublicFolderPath string   `yaml:"PublicFolderPath"`

	WatchViewsFolder bool `yaml:"WatchViewsFolder"`

	StaticAccessLog  bool
	DynamicAccessLog bool
	Verbose          bool
	GZipDynamic      bool
	GZipStatic       bool
	SessionDebug     bool `yaml:"SessionDebug"`

	StaticIndexFiles []string `yaml:"StaticIndexFiles"`

	//Gracefully restarts if enabled
	GracefulRestart bool
}

func (a *AppConfig) ParseEnv() {
	re := regexp.MustCompile("(\\$_*[A-Z][A-Z0-9_]+)")
	replacer := func(raw string) string {
		return os.Getenv(raw[1:])
	}
	a.Name = re.ReplaceAllStringFunc(a.Name, replacer)
	a.GlobalPageTitle = re.ReplaceAllStringFunc(a.GlobalPageTitle, replacer)
	a.Version = re.ReplaceAllStringFunc(a.Version, replacer)
	a.HostAddr = re.ReplaceAllStringFunc(a.HostAddr, replacer)
	a.HostAddrTLS = re.ReplaceAllStringFunc(a.HostAddrTLS, replacer)
	a.MongoDbs = re.ReplaceAllStringFunc(a.MongoDbs, replacer)
	a.Database = re.ReplaceAllStringFunc(a.Database, replacer)
	a.Salt = re.ReplaceAllStringFunc(a.Salt, replacer)
	a.LocalePath = re.ReplaceAllStringFunc(a.LocalePath, replacer)
	a.DefaultLanguage = re.ReplaceAllStringFunc(a.DefaultLanguage, replacer)
	//Data
	if a.Data != nil {
		for k, _ := range a.Data {
			a.Data[k] = re.ReplaceAllStringFunc(a.Data[k], replacer)
		}
	}
	//TLS
	a.TLSCertificatePath = re.ReplaceAllStringFunc(a.TLSCertificatePath, replacer)
	a.TLSKeyPath = re.ReplaceAllStringFunc(a.TLSKeyPath, replacer)

	// Paths
	a.RoutesConfigPath = re.ReplaceAllStringFunc(a.RoutesConfigPath, replacer)
	a.CachePath = re.ReplaceAllStringFunc(a.CachePath, replacer)
	a.ViewsFolderPath = re.ReplaceAllStringFunc(a.ViewsFolderPath, replacer)
	//TODO: maybe parse env vars on views extensions
	a.PublicFolderPath = re.ReplaceAllStringFunc(a.PublicFolderPath, replacer)
}
