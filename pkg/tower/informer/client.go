package informer

import (
	graphcclient "github.com/everoute/graphc/pkg/client"
	"github.com/everoute/graphc/pkg/crcwatch"

	"github.com/everoute/trafficredirect/pkg/config"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

func NewTowerClient() *graphcclient.Client {
	return &graphcclient.Client{
		URL: "https://" + config.Config.Tower.Addr + "/api",
		UserInfo: &graphcclient.UserInfo{
			Username: config.Config.Tower.Username,
			Password: config.Config.Tower.Password,
			Source:   config.Config.Tower.Source,
		},
		AllowInsecure: config.Config.Tower.AllowInsecure,
	}
}

func NewCRCWatch(resourceTypes []datamodel.ResourceType) (*crcwatch.Watch, error) {
	userInfo := &graphcclient.UserInfo{
		Username: config.Config.Tower.Username,
		Password: config.Config.Tower.Password,
		Source:   config.Config.Tower.Source,
	}
	resTypes := []string{}
	for _, t := range resourceTypes {
		resTypes = append(resTypes, string(t))
	}
	return crcwatch.NewWatch(resTypes, crcwatch.SetUserInfo(userInfo),
		crcwatch.SetAPIAuth(config.Config.Tower.APIUsername, config.Config.Tower.APIPassword),
		crcwatch.SetHost(config.Config.Tower.Addr))
}
