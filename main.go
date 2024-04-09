package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"

	ecr "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
)

func main() {

	var ECRRegistry string

	apiPort := flag.Int("port", 8080, "listen on this port")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.StringVar(&ECRRegistry, "ecr_registry", "", "ECR registry")
	flag.Parse()

	if ECRRegistry == "" {
		fmt.Println("ecr_registry must be set")
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
	_, _, err := ecrCredHelper.Get(ECRRegistry)
	if err != nil {
		log.Fatalf("error getting credentials %q", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		proxy := httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				username, password, err := ecrCredHelper.Get(ECRRegistry)
				if err != nil {
					slog.Error("error getting credentials", "err", err)
					return
				}
				r.Out.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

				r.Out.URL.Scheme = "https"
				r.Out.URL.Host = ECRRegistry
				r.Out.Host = ECRRegistry

				slog.InfoContext(r.In.Context(), "ECR Request", "Method", r.In.Method, "Path", r.In.URL.Path)
			},
			ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
				slog.ErrorContext(req.Context(), "proxy backend error", "err", err)
				rw.WriteHeader(http.StatusBadGateway)
			},
			ModifyResponse: func(resp *http.Response) error {
				slog.DebugContext(resp.Request.Context(), "ECR Response", "status", resp.Status, "header", resp.Header)
				return nil
			},
		}

		proxy.ServeHTTP(w, req)
	})

	fmt.Printf("Listening on port %d...\n", *apiPort)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *apiPort), mux))
}
