package goboots

import (
	"os"
	"regexp"
)

type AppConfig struct {
	Name            string
	Version         string
	HostAddr        string
	HostAddrTLS     string
	MongoDbs        string
	Database        string
	Salt            string
	LocalePath      string
	DefaultLanguage string
	Data            map[string]string

	// TLS
	TLSCertificatePath string
	TLSKeyPath         string
	TLSRedirect        bool

	// Paths
	RoutesConfigPath string
	ViewsFolderPath  string
	ViewsExtensions  []string // .html, .tpl
	PublicFolderPath string
}

func (a *AppConfig) ParseEnv() {
	re := regexp.MustCompile("(\\$_*[A-Z][A-Z0-9_]+)")
	replacer := func(raw string) string {
		return os.Getenv(raw[1:])
	}
	a.Name = re.ReplaceAllStringFunc(a.Name, replacer)
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
	a.ViewsFolderPath = re.ReplaceAllStringFunc(a.ViewsFolderPath, replacer)
	//TODO: maybe parse env vars on views extensions
	a.PublicFolderPath = re.ReplaceAllStringFunc(a.PublicFolderPath, replacer)
}
