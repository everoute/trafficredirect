package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	graphcclient "github.com/everoute/graphc/pkg/client"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/everoute/trafficredirect/pkg/config"
	"github.com/everoute/trafficredirect/pkg/tower/datamodel"
)

type Client struct {
	Cli *graphcclient.Client
}

func NewClient() *Client {
	cli := &graphcclient.Client{
		URL: "https://" + config.Config.Tower.Addr + "/api",
		UserInfo: &graphcclient.UserInfo{
			Username: config.Config.Tower.Username,
			Password: config.Config.Tower.Password,
			Source:   config.Config.Tower.Source,
		},
		AllowInsecure: config.Config.Tower.AllowInsecure,
	}
	if _, err := cli.Auth(); err != nil {
		ctrl.Log.Error(err, "tower client auth failed, new tower client failed")
		os.Exit(1)
	}
	return &Client{Cli: cli}
}

func (c *Client) Get(ctx context.Context, id string, obj datamodel.GqlType) (bool, error) {
	return c.get(ctx, id, obj, true)
}

func (c *Client) get(ctx context.Context, id string, obj datamodel.GqlType, authRetry bool) (bool, error) {
	log := ctrl.LoggerFrom(ctx)
	req := &graphcclient.Request{Query: obj.GqlGetStr(id)}
	resp, err := c.Cli.Query(req)
	if err != nil {
		log.Error(err, "query gql error")
		return false, err
	}
	if len(resp.Errors) > 0 {
		err := aggregateRespErrors(resp.Errors)
		log.Error(err, "query gql resp with errors")
		if authRetry && graphcclient.HasAuthError(resp.Errors) {
			_, err := c.Cli.Auth()
			if err != nil {
				log.Error(err, "tower client re-auth failed")
				return false, err
			}
			log.Info("tower client re-auth success")
			return c.get(ctx, id, obj, false)
		}
		return false, err
	}

	data := make(map[string]json.RawMessage)
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		log.Error(err, "unmarshal gql resp data error", "data", string(resp.Data))
		return false, err
	}
	if _, ok := data[obj.TypeName()]; !ok {
		log.Error(err, "gql resp data missing object", "object", obj.TypeName(), "data", data)
		return false, fmt.Errorf("gql resp data missing object %s", obj.TypeName())
	}
	if string(data[obj.TypeName()]) == "null" {
		return false, nil
	}
	if err := json.Unmarshal(data[obj.TypeName()], obj); err != nil {
		log.Error(err, "unmarshal gql resp object error", "object", obj.TypeName(), "data", data)
		return false, err
	}
	return true, nil
}

func aggregateRespErrors(errs []graphcclient.ResponseError) error {
	if len(errs) == 0 {
		return nil
	}
	errList := make([]error, 0, len(errs))
	for _, e := range errs {
		errList = append(errList, e)
	}
	return errors.NewAggregate(errList)
}
