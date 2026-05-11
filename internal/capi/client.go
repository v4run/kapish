package capi

import (
	"errors"
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Options configure NewClient.
type Options struct {
	Kubeconfig string // explicit path; empty falls back to $KUBECONFIG / ~/.kube/config
	Context    string // overrides kubeconfig's current-context
	Namespace  string // narrows list/watch; empty = all namespaces
}

// Client is kapish's wrapper over client-go for talking to a CAPI mgmt cluster.
type Client struct {
	dyn       dynamic.Interface
	kube      kubernetes.Interface
	rest      *rest.Config
	context   string
	namespace string
}

// NewClient resolves kubeconfig+context, builds clients, and returns the wrapper.
func NewClient(opts Options) (*Client, error) {
	loader := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		loader.ExplicitPath = opts.Kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)

	rawCfg, err := cfg.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("capi: load kubeconfig: %w", err)
	}
	ctxName := opts.Context
	if ctxName == "" {
		ctxName = rawCfg.CurrentContext
	}
	if ctxName == "" {
		return nil, errors.New("capi: kubeconfig has no current-context and none specified")
	}
	if _, ok := rawCfg.Contexts[ctxName]; !ok {
		return nil, fmt.Errorf("capi: context %q not found in kubeconfig", ctxName)
	}

	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("capi: build REST config: %w", err)
	}
	dyn, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("capi: build dynamic client: %w", err)
	}
	kube, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("capi: build kubernetes client: %w", err)
	}

	return &Client{
		dyn:       dyn,
		kube:      kube,
		rest:      restCfg,
		context:   ctxName,
		namespace: opts.Namespace,
	}, nil
}

// Context reports the resolved kubeconfig context this Client uses.
func (c *Client) Context() string { return c.context }

// Namespace reports the namespace scope ("" = all).
func (c *Client) Namespace() string { return c.namespace }
