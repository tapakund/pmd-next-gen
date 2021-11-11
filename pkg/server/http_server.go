// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/pm-web/pkg/conf"
	"github.com/pm-web/pkg/share"
	"github.com/pm-web/pkg/system"
	"github.com/pm-web/plugins/proc"
	"github.com/pm-web/plugins/systemd"
	"github.com/pm-web/plugins/management"
)

var httpSrv *http.Server

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	s := r.PathPrefix("/api/v1").Subrouter()

	systemd.InitSystemd()
	systemd.RegisterRouterSystemd(s)
	management.RegisterRouterManagement(s)
	proc.RegisterRouterProc(s)

	return r
}

func runUnixDomainHttpServer(c *conf.Config, r *mux.Router) error {
	var credentialsContextKey = struct{}{}

	if c.System.UseAuthentication {
		r.Use(UnixDomainPeerCredential)
	}

	httpSrv = &http.Server{
		Handler: r,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			file, _ := c.(*net.UnixConn).File()
			credentials, _ := unix.GetsockoptUcred(int(file.Fd()), unix.SOL_SOCKET, unix.SO_PEERCRED)
			return context.WithValue(ctx, credentialsContextKey, credentials)
		},
	}

	log.Infof("Starting pm-webd server at unix domain socket='%s' in HTTP mode", conf.UnixDomainSocketPath)

	os.Remove(conf.UnixDomainSocketPath)
	unixListener, err := net.ListenUnix("unix", &net.UnixAddr{Name: conf.UnixDomainSocketPath, Net: "unix"})
	if err != nil {
		log.Fatalf("Unable to listen on unix domain socket='%s': %v", conf.UnixDomainSocketPath, err)
	}
	defer unixListener.Close()

	if err := system.ChangeUnixDomainSocketPermission(conf.UnixDomainSocketPath); err != nil {
		log.Errorf("Failed to change unix domain socket permissions: %v", err)
		return err
	}

	log.Fatal(httpSrv.Serve(unixListener))

	return nil
}

func runWebHttpServer(c *conf.Config, r *mux.Router) error {
	if c.System.UseAuthentication {
		amw, err := InitAuthMiddleware()
		if err != nil {
			log.Fatalf("Failed to init auth DB existing: %v", err)
			return err
		}

		r.Use(amw.AuthMiddleware)
	}

	ip, port, _ := share.ParseIpPort(c.Network.Listen)

	if system.TLSFilePathExits() {
		cfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: false,
		}
		httpSrv = &http.Server{
			Addr:         ip + ":" + port,
			Handler:      r,
			TLSConfig:    cfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}

		log.Infof("Starting pm-webd server at %s:%s in HTTPS mode", ip, port)

		log.Fatal(httpSrv.ListenAndServeTLS(path.Join(conf.ConfPath, conf.TLSCert), path.Join(conf.ConfPath, conf.TLSKey)))
	} else {
		httpSrv := http.Server{
			Addr:    ip + ":" + port,
			Handler: r,
		}

		log.Infof("Starting pm-webd server at %s:%s in HTTP mode", ip, port)

		log.Fatal(httpSrv.ListenAndServe())
	}

	return nil
}

func Run(c *conf.Config) error {
	r := NewRouter()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		if err := httpSrv.Shutdown(context.Background()); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()

	if c.Network.ListenUnixSocket {
		runUnixDomainHttpServer(c, r)
	} else {
		runWebHttpServer(c, r)
	}

	return nil
}
