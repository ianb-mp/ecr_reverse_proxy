package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
)

func main() {

	apiPort := flag.Int("port", 8080, "listen on this port")
	debug := flag.Bool("debug", false, "enable debug logging")
	ecrRegistry := flag.String("ecr_registry", "", "ECR registry")
	proxyHostname := flag.String("proxy_hostname", "", "proxy hostname")
	flag.Parse()

	if *ecrRegistry == "" {
		fmt.Println("ecr_registry must be set")
		os.Exit(1)
	}
	if *proxyHostname == "" {
		fmt.Println("proxy_hostname must be set")
		os.Exit(1)
	}

	var programLevel = new(slog.LevelVar) // Info by default
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)
	if *debug {
		programLevel.Set(slog.LevelDebug)
	}

	ecrCredHelper := ecr.NewECRHelper()

	// Attempt to fetch credentials initially
	_, _, err := ecrCredHelper.Get(*ecrRegistry)
	if err != nil {
		log.Fatalf("error getting credentials %q", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		proxy := httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				username, password, err := ecrCredHelper.Get(*ecrRegistry)
				if err != nil {
					slog.Error("error getting credentials", "err", err)
					return
				}
				r.Out.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

				r.Out.URL.Scheme = "https"
				r.Out.URL.Host = *ecrRegistry
				r.Out.Host = *ecrRegistry

				slog.InfoContext(r.In.Context(), "ECR Request", "Method", r.In.Method, "Path", r.In.URL.Path)
			},
			ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
				slog.ErrorContext(req.Context(), "proxy backend error", "err", err)
				rw.WriteHeader(http.StatusBadGateway)
			},
			ModifyResponse: func(resp *http.Response) error {
				slog.DebugContext(resp.Request.Context(), "ECR Response", "status", resp.Status, "header", resp.Header)

				// if ECR returns a Location header, then swap out the ECR hostname for the proxy hostname:port
				if resp.Header.Get("Location") != "" {
					redir, err := url.Parse(resp.Header.Get("Location"))
					if err != nil {
						return err
					}

					newRedir := *redir
					newRedir.Scheme = "http"
					newRedir.Host = fmt.Sprintf("%s:%d", *proxyHostname, *apiPort)

					slog.Debug("rewrite ECR redirect", "from", redir.String(), "to", newRedir.String())
					resp.Header.Set("Location", newRedir.String())
				}
				return nil
			},
		}

		proxy.ServeHTTP(w, req)
	})

	fmt.Printf("Listening on port %d...\n", *apiPort)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *apiPort), mux))
}
