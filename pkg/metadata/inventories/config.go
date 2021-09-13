package inventories

import "github.com/DataDog/datadog-agent/pkg/config"

// SetConfigMetadata sets the agent metadata based on the given configuration
func SetConfigMetadata(cfg config.Config) {
	SetAgentMetadata("config_apm_dd_url", cfg.GetString("apm_config.dd_url"))
	SetAgentMetadata("config_dd_url", cfg.GetString("dd_url"))
	SetAgentMetadata("config_logs_dd_url", cfg.GetString("logs_config.logs_dd_url"))
	SetAgentMetadata("config_logs_socks5_proxy_address", cfg.GetString("logs_config.socks5_proxy_address"))
	SetAgentMetadata("config_no_proxy", cfg.GetStringSlice("proxy.no_proxy"))
	SetAgentMetadata("config_process_dd_url", cfg.GetString("process_config.process_dd_url"))
	SetAgentMetadata("config_proxy_http", cfg.GetString("proxy.http"))
	SetAgentMetadata("config_proxy_https", cfg.GetString("proxy.https"))
}
