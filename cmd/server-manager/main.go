package main

import (
	"net"
	"net/http"
	"os"
	"runtime"

	"github.com/cj123/assetto-server-manager"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/static"
	"github.com/cj123/assetto-server-manager/cmd/server-manager/views"
	"github.com/cj123/assetto-server-manager/pkg/udp"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

var debug = os.Getenv("DEBUG") == "true"

func main() {
	config, err := servermanager.ReadConfig("config.yml")

	if err != nil {
		logrus.Fatalf("could not open config file, err: %s", err)
	}

	store, err := config.Store.BuildStore()

	if err != nil {
		logrus.Fatalf("could not open store, err: %s", err)
	}

	servermanager.SetupRaceManager(store)
	servermanager.SetAssettoInstallPath(config.Steam.InstallPath)

	err = servermanager.InstallAssettoCorsaServer(config.Steam.Username, config.Steam.Password, config.Steam.ForceUpdate)

	if err != nil {
		logrus.Fatalf("could not install assetto corsa server, err: %s", err)
	}

	var templateLoader servermanager.TemplateLoader
	var filesystem http.FileSystem

	if os.Getenv("FILESYSTEM_HTML") == "true" {
		templateLoader = servermanager.NewFilesystemTemplateLoader("views")
		filesystem = http.Dir("static")
	} else {
		templateLoader = &views.TemplateLoader{}
		filesystem = static.FS(false)
	}

	if config.LiveMap.IsEnabled() {
		udp.RealtimePosIntervalMs = config.LiveMap.IntervalMs
	}

	servermanager.ViewRenderer, err = servermanager.NewRenderer(templateLoader, debug)

	if err != nil {
		logrus.Fatalf("could not initialise view renderer, err: %s", err)
	}

	go servermanager.LoopRaces()
	err = servermanager.InitialiseScheduledCustomRaces()

	if err != nil {
		logrus.Errorf("couldn't initialise scheduled races, err: %s", err)
	}

	err = servermanager.InitialiseScheduledChampionshipEvents()

	if err != nil {
		logrus.Errorf("couldn't initialise scheduled championship events, err: %s", err)
	}

	listener, err := net.Listen("tcp", config.HTTP.Hostname)

	if err != nil {
		logrus.Fatalf("could not listen on address: %s, err: %s", config.HTTP.Hostname, err)
	}

	logrus.Infof("starting assetto server manager on: %s", config.HTTP.Hostname)

	if runtime.GOOS == "windows" {
		_ = browser.OpenURL("http://" + config.HTTP.Hostname)
	}

	if err := http.Serve(listener, servermanager.Router(filesystem)); err != nil {
		logrus.Fatal(err)
	}
}
