package geoserver

import (
	"GH-Server/pkg/zaplog"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

const (
	STATEMENT = "GEOSERVER"
)

var(
	geoserAddress = os.Getenv("GEOSERVER_ADDRESS")
	geoserUsername = os.Getenv("GEOSERVER_USER")
	geoserPassword = os.Getenv("GEOSERVER_PASS")
	geoserWorkspace = os.Getenv("GEOSERVER_WORKSPACE")
	dbHost                  = os.Getenv("DB_HOST")
	dbPort,_                  = strconv.ParseInt(os.Getenv("DB_PORT"),10,64)
	dbUser                  = os.Getenv("DB_USER")
	dbPassword              = os.Getenv("DB_PASS")
	dbName                  = os.Getenv("GEOSERVER_DB_NAME")
)

type GeoServerService interface {
}
type geoServerService struct {
	*geoserver.Client
}

func New() GeoServerService{
	ctx := context.Background()
	client,err := geoserver.New(geoserAddress,
	geoserver.WithBasicAuth(geoserUsername, geoserPassword),
	geoserver.WithTimeout(10*time.Second),)
	if err != nil {
		zaplog.Zap.Panic("GeoServer客户端启动失败")
	}
	err = client.Workspaces.Create(ctx,&workspaces.Workspace{Name:geoserWorkspace })
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("创建工作空间失败:%s", err.Error()))
	}
	if err := client.Datastores.InWorkspace(geoserWorkspace).Create(ctx,datastores.PostGIS{
		Name: geoserWorkspace,
		Host: dbHost,
		Port: int(dbPort),
		User: dbUser,
		Password: dbPassword,
		Database: dbName,
	}); err != nil {
		zaplog.Zap.Panic(fmt.Sprintf("POSTGIS数据源创建失败:%s",err.Error()))
	}
	return  &geoServerService{
		Client: client,
	}
}