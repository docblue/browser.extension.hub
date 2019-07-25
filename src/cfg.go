package main

import (
	"gopkg.in/ini.v1"
)

func parseConfigurationFile(cfgFilename string) error {
	cfg, err := ini.LoadSources(ini.LoadOptions{Insensitive: true, AllowShadows: false}, cfgFilename)
	if err == nil {
		cgiUpdateUrl = cfg.Section("").Key("cgi.update.url").String()

		if rawPort, _ := cfg.Section("").Key("listen.port").Int(); rawPort > 100 {
			listenningPort = rawPort
		}

		if rawRpcTimeout, _ := cfg.Section("").Key("rpc.response.timeoutt").Int(); rawRpcTimeout >= 3 {
			fileWatcherTimeout = rawRpcTimeout
		}
	}

	return err
}

func init() {

}
