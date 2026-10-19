package configs

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/spf13/viper"
)

type TlsConfig interface {
	GetProvider() string
	GetProp(key string) (string, bool)
}

type Config interface {
	GetBindIPAddress() string
	GetReleaseSyncInterval() time.Duration
	GetCommandCheckInterval() time.Duration
	GetCertificateCheckInterval() time.Duration

	GetDownloadDir() string
	GetDataDir() string

	GetCertificatesDir() string
	GetAccountsDir() string

	GetMinCertificateTtl() time.Duration
	GetMaxDomainCertAttempts() int
	GetCertRequestPlannerInterval() time.Duration
	GetCertRequestExecutorInterval() time.Duration

	GetDomain() string
	GetPublicURL() string

	GetListenIPAddress() string
	GetHttpPort() string

	IsHttpsEnabled() bool
	IsHttpsRedirectDisabled() bool

	GetHttpsPort() string
	GetAcmeEmail() string

	GetTlsConfig() TlsConfig
}

type tls_configs struct {
	Provider string            `mapstructure:"provider" yaml:"provider"`
	Props    map[string]string `mapstructure:"props" yaml:"props"`
}

var _ TlsConfig = (*tls_configs)(nil)

func (c *tls_configs) GetProvider() string {
	provider := strings.TrimSpace(c.Provider)
	if provider == "" {
		slog.Warn("TLS provider is empty, using default 'selfsigned'")
		return "selfsigned"
	}
	return provider
}

func (c *tls_configs) GetProp(key string) (string, bool) {
	if c.Props == nil {
		return "", false
	}
	val, ok := c.Props[key]
	return val, ok
}

type configs struct {
	BindAddress              string `mapstructure:"bind_address" yaml:"bind_address"`                             // default: 127.0.0.1
	ReleaseSyncInterval      string `mapstructure:"release_sync_interval" yaml:"release_sync_interval"`           // default: 10m
	CommandCheckInterval     string `mapstructure:"command_check_interval" yaml:"command_check_interval"`         // default: 10ms
	CertificateCheckInterval string `mapstructure:"certificate_check_interval" yaml:"certificate_check_interval"` // default: 1h

	DownloadDir string `mapstructure:"download_dir" yaml:"download_dir"` // default: ./downloads

	CertificatesDir string `mapstructure:"certificates_dir" yaml:"certificates_dir"` // default: ./.certificates
	AccountsDir     string `mapstructure:"accounts_dir" yaml:"accounts_dir"`         // default: ./.accounts

	DataDir   string `mapstructure:"data_dir" yaml:"data_dir"` // default: ./data
	Domain    string `mapstructure:"domain" yaml:"domain"`
	PublicURL string `mapstructure:"public_url" yaml:"public_url"`

	ListenAddress               string `mapstructure:"listen_address" yaml:"listen_address"` // default: 0.0.0.0
	HttpPort                    string `mapstructure:"http_port" yaml:"http_port"`           // default: 8072
	Https                       bool   `mapstructure:"https" yaml:"https"`
	DisableHttpsRedirect        bool   `mapstructure:"disable_https_redirect" yaml:"disable_https_redirect"`
	HttpsPort                   string `mapstructure:"https_port" yaml:"https_port"`                                         // default: 8443
	MinCertificateTtl           string `mapstructure:"min_certificate_ttl" yaml:"min_certificate_ttl"`                       // default: 720h
	MaxDomainCertAttempts       int    `mapstructure:"max_domain_cert_attempts" yaml:"max_domain_cert_attempts"`             // default: 3
	CertRequestPlannerInterval  string `mapstructure:"cert_request_planner_interval" yaml:"cert_request_planner_interval"`   // default: 5m
	CertRequestExecutorInterval string `mapstructure:"cert_request_executor_interval" yaml:"cert_request_executor_interval"` // default: 1m

	AcmeEmail string `mapstructure:"acme_email" yaml:"acme_email"`

	Tls tls_configs `mapstructure:"cert" yaml:"cert"`
}

var _ Config = (*configs)(nil)

func (c *configs) GetBindIPAddress() string {
	if c.BindAddress == "" {
		return "127.0.0.1"
	}
	return c.BindAddress
}

const min_sync_interval = 5 * time.Minute
const min_command_check_interval = 10 * time.Second
const min_certificate_check_interval = time.Minute
const min_certificate_ttl = 30 * 24 * time.Hour
const min_cert_request_planner_interval = 5 * time.Minute
const min_cert_request_executor_interval = time.Minute

func (c *configs) GetReleaseSyncInterval() time.Duration {
	return parseDurationWithMin(
		c.ReleaseSyncInterval,
		min_sync_interval,
		"release_sync_interval",
	)
}

func (c *configs) GetCommandCheckInterval() time.Duration {
	return parseDurationWithMin(
		c.CommandCheckInterval,
		min_command_check_interval,
		"command_check_interval",
	)
}

func (c *configs) GetCertificateCheckInterval() time.Duration {
	return parseDurationWithMin(
		c.CertificateCheckInterval,
		min_certificate_check_interval,
		"certificate_check_interval",
	)
}

func (c *configs) GetDownloadDir() string {
	if c.DownloadDir == "" {
		return "./downloads"
	}
	return c.DownloadDir
}

func (c *configs) GetDataDir() string {
	if c.DataDir == "" {
		return "./data"
	}
	return c.DataDir
}

func (c *configs) GetCertificatesDir() string {
	if c.CertificatesDir == "" {
		return "./.certificates"
	}
	return c.CertificatesDir
}

func (c *configs) GetAccountsDir() string {
	if c.AccountsDir == "" {
		return "./.accounts"
	}
	return c.AccountsDir
}

func (c *configs) GetDomain() string {
	if c.Domain == "" {
		return "pb.labenv.test"
	}
	return strings.Split(c.Domain, ":")[0]
}

// GetPublicURL only advertises links; it never enables internal TLS or trusts
// client-supplied forwarding headers.
func (c *configs) GetPublicURL() string { return c.PublicURL }

func (c *configs) GetListenIPAddress() string {
	if c.ListenAddress == "" {
		return "0.0.0.0"
	}
	return c.ListenAddress
}

func (c *configs) GetHttpPort() string {
	if c.HttpPort == "" {
		return "7080"
	}
	return c.HttpPort
}

func (c *configs) IsHttpsEnabled() bool          { return c.Https }
func (c *configs) IsHttpsRedirectDisabled() bool { return c.DisableHttpsRedirect }

func (c *configs) GetHttpsPort() string {
	if c.HttpsPort == "" {
		return "8443"
	}
	return c.HttpsPort
}

func (c *configs) GetAcmeEmail() string {
	return strings.TrimSpace(c.AcmeEmail)
}

func (c *configs) GetMinCertificateTtl() time.Duration {
	return parseDurationWithMin(
		c.MinCertificateTtl,
		min_certificate_ttl,
		"min_certificate_ttl",
	)
}

func (c *configs) GetMaxDomainCertAttempts() int {
	if c.MaxDomainCertAttempts < 0 {
		slog.Warn("max_domain_cert_attempts below minimum, forcing to 1")
		return 1
	}
	if c.MaxDomainCertAttempts > 5 {
		slog.Warn("max_domain_cert_attempts above maximum, forcing to 5")
		return 5
	}
	return max(c.MaxDomainCertAttempts, 1)
}

func (c *configs) GetCertRequestPlannerInterval() time.Duration {
	return parseDurationWithMin(
		c.CertRequestPlannerInterval,
		min_cert_request_planner_interval,
		"cert_request_planner_interval",
	)
}

func (c *configs) GetCertRequestExecutorInterval() time.Duration {
	return parseDurationWithMin(
		c.CertRequestExecutorInterval,
		min_cert_request_executor_interval,
		"cert_request_executor_interval",
	)
}

func (c *configs) GetTlsConfig() TlsConfig { return &c.Tls }

func loadConfigFromFile(filePath string) (*configs, error) {
	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")

	var cfg configs

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, nil
		}
		slog.Error("failed to read config file", "file", path.Base(filePath), "error", err)
		return nil, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		slog.Error("failed to unmarshal config", "file", path.Base(filePath), "error", err)
		return nil, err
	}
	slog.Info("Loaded config file", slog.String("file_path", filePath))
	return &cfg, nil
}

func LoadConfigs(configPath string) (Config, error) {
	if configPath == "" {
		return &configs{}, nil
	}
	c, err := loadConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	c.ReleaseSyncInterval = strings.TrimSpace(c.ReleaseSyncInterval)
	c.DownloadDir = strings.TrimSpace(c.DownloadDir)
	c.DataDir = strings.TrimSpace(c.DataDir)
	c.Domain = strings.TrimSpace(c.Domain)
	c.BindAddress = strings.TrimSpace(c.BindAddress)

	c.ListenAddress = strings.TrimSpace(c.ListenAddress)
	c.HttpPort = strings.TrimSpace(c.HttpPort)
	c.HttpsPort = strings.TrimSpace(c.HttpsPort)
	c.AcmeEmail = strings.TrimSpace(c.AcmeEmail)

	if err := is.IPv4.Validate(c.GetBindIPAddress()); err != nil {
		slog.Error(
			"Invalid bind_address: not a valid IPv4 address",
			"value", c.GetBindIPAddress(),
		)
		return nil, fmt.Errorf("invalid bind_address: %w", err)
	}

	if err := is.IPv4.Validate(c.GetListenIPAddress()); err != nil {
		slog.Error(
			"Invalid listen_address: not a valid IPv4 address",
			"value", c.GetListenIPAddress(),
		)
		return nil, fmt.Errorf("invalid listen_address: %w", err)
	}

	c.PublicURL = strings.TrimRight(strings.TrimSpace(c.PublicURL), "/")
	if c.PublicURL != "" {
		if err := validatePublicURL(c.PublicURL, c.GetDomain()); err != nil {
			return nil, err
		}
	}
	portNum, err := strconv.Atoi(c.GetHttpPort())
	if err != nil || portNum < 1 || portNum > 65535 {
		return nil, errors.New("invalid http_port: must be an integer between 1 and 65535")
	}

	if c.IsHttpsEnabled() {
		if err := is.EmailFormat.Validate(c.AcmeEmail); err != nil {
			return nil, fmt.Errorf("invalid ACME email address: %w", err)
		}
		portNum, err := strconv.Atoi(c.GetHttpsPort())
		if err != nil || portNum < 1 || portNum > 65535 {
			return nil, errors.New("invalid https_port: must be an integer between 1 and 65535")
		}
	}
	return c, nil
}

func parseDurationWithMin(raw string, min time.Duration, name string) time.Duration {
	if raw == "" {
		return min
	}
	duration, err := time.ParseDuration(raw)
	if err != nil {
		slog.Warn("Failed to parse "+name,
			slog.String("raw_value", raw),
			slog.String("error", err.Error()),
		)
		return min
	}
	if duration < min {
		slog.Warn("Provided "+name+" is below minimum; using minimum instead",
			slog.String("raw_value", raw),
			slog.String("min_value", min.String()),
		)
		return min
	}
	return duration
}

func validatePublicURL(raw, domain string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") ||
		!strings.EqualFold(u.Hostname(), domain) || u.User != nil || u.Path != "" ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return errors.New("public_url must be an http(s) origin matching domain, without credentials, path, query or fragment")
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return errors.New("invalid public_url port")
		}
	}
	return nil
}
