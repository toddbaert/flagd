package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/open-feature/flagd/pkg/model"
	"github.com/open-feature/flagd/pkg/provider"
	"github.com/open-feature/flagd/pkg/runtime"
	"github.com/open-feature/flagd/pkg/service"
	"github.com/spf13/cobra"
)

var (
	serviceProvider   string
	syncProvider      string
	uri               string
	httpServicePort   int32
	socketServicePath string
)

func findService(name string) (service.IService, error) {
	registeredServices := map[string]service.IService{
		"http": &service.HttpService{
			HttpServiceConfiguration: &service.HttpServiceConfiguration{
				Port: int32(httpServicePort),
			},
		},
	}
	if v, ok := registeredServices[name]; !ok {
		return nil, errors.New("no service-provider set")
	} else {
		log.Debugf("Using %s service-provider\n", name)
		return v, nil
	}
}

func findProvider(name string) (provider.IProvider, error) {
	registeredSync := map[string]provider.IProvider{
		"filepath": &provider.FilePathProvider{
			URI: uri,
			Flags: model.Flags{},
		},
	}
	if v, ok := registeredSync[name]; !ok {
		return nil, errors.New("no sync-provider set")
	} else {
		log.Debugf("Using %s sync-provider\n", name)
		return v, nil
	}
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start flagd",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		// Configure provider impl--------------------------------------------
		var providerImpl provider.IProvider
		if foundSync, err := findProvider(syncProvider); err != nil {
			return
		} else {
			providerImpl = foundSync
		}

		// Configure service-provider impl------------------------------------------
		var serviceImpl service.IService
		if foundService, err := findService(serviceProvider); err != nil {
			return
		} else {
			serviceImpl = foundService
		}

		// Serve ------------------------------------------------------------------
		ctx, cancel := context.WithCancel(context.Background())
		errc := make(chan error)
		go func() {
			errc <- func() error {
				c := make(chan os.Signal)
				signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
				return fmt.Errorf("%s", <-c)
			}()
		}()

		go runtime.Start(serviceImpl, providerImpl, ctx)

		err := <-errc
		if err != nil {
			cancel()
			log.Printf(err.Error())
		}
	},
}

func init() {
	startCmd.Flags().Int32VarP(&httpServicePort, "port", "p", 8080, "Port to listen on")
	startCmd.Flags().StringVarP(&socketServicePath, "socketpath", "d", "/tmp/flagd.sock", "flagd socket path")
	startCmd.Flags().StringVarP(&serviceProvider, "service-provider", "s", "http", "Set a serve provider e.g. http or socket")
	startCmd.Flags().StringVarP(&syncProvider, "sync-provider", "y", "filepath", "Set a sync provider e.g. filepath or remote")
	startCmd.Flags().StringVarP(&uri, "uri", "f", "", "Set a sync provider uri to read data from this can be a filepath or url")
	rootCmd.AddCommand(startCmd)

}
