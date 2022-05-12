package ociutil

import (
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/pkg/oras"
)

func FetchertoProvider(fetcher remotes.Fetcher) content.Provider {
	if prov, ok := fetcher.(content.Provider); ok {
		log.Debugf("fetcher %T is a provider", fetcher)
		return prov
	}

	return &oras.ProviderWrapper{
		Fetcher: fetcher,
	}
}
