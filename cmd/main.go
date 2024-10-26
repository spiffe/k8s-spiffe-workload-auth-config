package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"gopkg.in/yaml.v3"
)

type watcher struct {
	trustDomain string
	url string
	source string
	dest string
}

func (w *watcher) OnX509BundlesUpdate(bs *x509bundle.Set) {
	var bundle *x509bundle.Bundle
	if w.trustDomain != "" {
		td, _ := spiffeid.TrustDomainFromString(w.trustDomain)
		bundle, _ = bs.GetX509BundleForTrustDomain(td)
	} else {
		if bs.Len() != 1 {
			log.Printf("Trust bundle doesn't have just one trust domain. You must specify the trust domain.")
			os.Exit(1)
		}
		bundle = bs.Bundles()[0]
	}
	b, _ := bundle.Marshal()
	log.Printf("Got new bundle")
	updateConfig(w.url, string(b), w.source, w.dest)
}

func (w *watcher) OnX509BundlesWatchError(err error) {
	log.Fatal(err)
}

func main() {
	var dest string
	trustDomain, _ := os.LookupEnv("SPIFFE_TRUST_DOMAIN")
	socketPath, ok := os.LookupEnv("SPIFFE_ENDPOINT_SOCKET")
	if !ok {
		socketPath = "unix:///tmp/spire-agent/public/api.sock"
	}
	l := len(os.Args)
	if l < 2 || l > 3 {
		fmt.Printf("Usage:\n")
		fmt.Printf("  %s <sourcefile> destfile\n", os.Args[0])
		os.Exit(1)
	}
	source := os.Args[1]
	dest = os.Args[1]
	if l == 3 {
		dest = os.Args[2]
	}
	ctx := context.Background()
	c, err := workloadapi.New(ctx, workloadapi.WithAddr(socketPath))
	if err != nil {
		log.Fatal(err)
	}
	svid, err := c.FetchJWTSVID(ctx, jwtsvid.Params{
		Audience: "k8s",
	})
	if err != nil {
		log.Fatal(err)
	}
	iss, ok := svid.Claims["iss"]
	if !ok || iss.(string) == "" {
		log.Fatal("Your server does not have a jwt_issuer configured. That is a required setting.")
	}
	isss := iss.(string)
	log.Printf("Issuer: %s\n", isss)

	w := &watcher{
		trustDomain: trustDomain,
		url: isss,
		source: source,
		dest: dest,
	}
	err = c.WatchX509Bundles(ctx, w)
	if err != nil {
		log.Fatal(err)
	}
}

func updateConfig(url string, cas string, source string, dest string) {
	b, err := os.ReadFile(source)
	if err != nil {
		log.Fatalf("Problem opening file: %v", err)
	}
	urlFound := false
	var config yaml.Node
	var caRef *yaml.Node
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		log.Fatalf("Problem unmarshaling config: %v", err)
	}
	for _, d := range config.Content {
		jwtFound := false
		for _, j := range d.Content {
			if jwtFound {
				for _, item := range j.Content {
					issuerFound := false
					for _, issuer := range item.Content {
						if issuerFound {
							for idx, isetting := range issuer.Content {
								if isetting.Value == "url" {
									if issuer.Content[idx + 1].Value == url {
										log.Printf("Found issuer")
										urlFound = true
									}
								}
								if  isetting.Value == "certificateAuthority" {
									log.Printf("Updating CAs")
									caRef = issuer.Content[idx + 1]
									caRef.SetString(cas)
								}
							}
							if urlFound && caRef == nil {
								log.Printf("Updating CAs")
								key := &yaml.Node{}
								key.SetString("certificateAuthority")
								value := &yaml.Node{}
								value.SetString(cas)
								issuer.Content = append(issuer.Content, key)
								issuer.Content = append(issuer.Content, value)
								caRef = value
							}
							break
						}
						if issuer.Value == "issuer" {
							issuerFound = true
						}
					}
				}
				break
			}
			if j.Value == "jwt" {
				jwtFound = true
			}
		}
	}
	f, err := os.CreateTemp(filepath.Dir(dest), "modified")
	if err != nil {
		log.Fatalf("Problem creating file: %v", err)
	}
	enc := yaml.NewEncoder(f)
	err = enc.Encode(config.Content[0])
	if err != nil {
		log.Fatalf("Failed to encode content: %v", err)
	}
	name := f.Name()
	f.Close()
	err = os.Rename(name, dest)
	if err != nil {
		log.Fatalf("Failed to rename file into place: %v", err)
	}
}
