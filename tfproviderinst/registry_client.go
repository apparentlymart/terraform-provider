package tfproviderinst

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/apparentlymart/go-versions/versions"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"
	disco "github.com/hashicorp/terraform-svchost/disco"
)

// RegistryClient is a client for the Terraform provider registry protocol.
type RegistryClient struct {
	services      *disco.Disco
	httpTransport http.RoundTripper
	httpClient    *http.Client
}

// NewRegistryClient constructs a new registry client with the given options.
//
// With no options given, the client will perform service discovery in the
// default way and it will not have access to any host-specific credentials.
// To interact with private registries, pass a [RegistryClientServices] result
// whose discovery object has a credentials source.
func NewRegistryClient(opts ...RegistryClientOption) *RegistryClient {
	ret := &RegistryClient{}
	for _, opt := range opts {
		opt.configureClient(ret)
	}

	if ret.services == nil {
		ret.services = disco.New()
	}

	ret.httpClient = &http.Client{
		Transport: ret.httpTransport,
	}

	return ret
}

// RegistryClientOption represents an option for [NewRegistryClient].
type RegistryClientOption struct {
	configureClient func(*RegistryClient)
}

// RegistryClientServices constructs a registry client option which provides
// a custom service discovery object.
func RegistryClientServices(services *disco.Disco) RegistryClientOption {
	return RegistryClientOption{
		configureClient: func(rc *RegistryClient) {
			if rc.services != nil {
				panic("multiple RegistryClientServices options")
			}
			rc.services = services
		},
	}
}

// RegistryClientServices constructs a registry client option which provides
// a custom HTTP "round-tripper".
func RegistryClientHTTPTransport(transport http.RoundTripper) RegistryClientOption {
	return RegistryClientOption{
		configureClient: func(rc *RegistryClient) {
			if rc.httpTransport != nil {
				panic("multiple RegistryClientHTTPTransport options")
			}
			rc.httpTransport = transport
		},
	}
}

// ProviderVersions asks the origin registry for the given provider which
// versions have available packages, and returns the version information
// for those packages.
func (c *RegistryClient) ProviderVersions(ctx context.Context, addr tfaddr.Provider) (found []*ProviderVersion, warnings []string, err error) {
	baseURL, creds, err := c.disco(ctx, addr.Hostname)
	if err != nil {
		return nil, nil, fmt.Errorf("service discovery failed for host %s: %w", addr.Hostname, err)
	}

	endpointPath, err := url.Parse(path.Join(addr.Namespace, addr.Type, "versions"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct endpoint path: %w", err)
	}

	endpointURL := baseURL.ResolveReference(endpointPath)
	req, err := http.NewRequestWithContext(ctx, "GET", endpointURL.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to construct request: %w", err)
	}
	creds.PrepareRequest(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Great!
	case http.StatusNotFound:
		return nil, nil, fmt.Errorf("registry %s does not host a provider named %s", addr.Hostname, addr)
	default:
		return nil, nil, fmt.Errorf("registry request failed: %s", resp.Status)
	}

	type ResponseBody struct {
		Versions []struct {
			Version   string   `json:"version"`
			Protocols []string `json:"protocols"`
			Platforms []struct {
				OS   string `json:"os"`
				Arch string `json:"arch"`
			} `json:"platforms"`
		} `json:"versions"`
		Warnings []string `json:"warnings"`
	}
	var body ResponseBody

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return nil, nil, fmt.Errorf("registry returned invalid response: %w", err)
	}

	if len(body.Versions) == 0 {
		return nil, body.Warnings, nil
	}

	ret := make([]*ProviderVersion, 0, len(body.Versions))
	for _, v := range body.Versions {
		version, err := versions.ParseVersion(v.Version)
		if err != nil {
			return nil, body.Warnings, fmt.Errorf("registry returned invalid version string %q: %w", v.Version, err)
		}
		retV := &ProviderVersion{
			Provider:           addr,
			Version:            version,
			SupportedProtocols: make([]ProtocolVersion, 0, len(v.Protocols)),
			SupportedPlatforms: make([]Platform, 0, len(v.Platforms)),
		}
		sort.Strings(v.Protocols)
		sort.Slice(v.Platforms, func(i, j int) bool {
			switch {
			case v.Platforms[i].OS != v.Platforms[j].OS:
				return v.Platforms[i].OS < v.Platforms[j].OS
			default:
				return v.Platforms[i].Arch < v.Platforms[j].Arch
			}
		})
		for _, rawPV := range v.Protocols {
			pv, err := ParseProtocolVersion(rawPV)
			if err != nil {
				return nil, body.Warnings, fmt.Errorf("registry returned invalid protocol version string %q: %w", rawPV, err)
			}
			retV.SupportedProtocols = append(retV.SupportedProtocols, pv)
		}
		for _, rawPlatform := range v.Platforms {
			if !validPlatformPart(rawPlatform.OS) {
				return nil, body.Warnings, fmt.Errorf("registry returned invalid OS name %q", rawPlatform.OS)
			}
			if !validPlatformPart(rawPlatform.Arch) {
				return nil, body.Warnings, fmt.Errorf("registry returned invalid architecture name %q", rawPlatform.Arch)
			}
			retV.SupportedPlatforms = append(retV.SupportedPlatforms, Platform{rawPlatform.OS, rawPlatform.Arch})
		}

		ret = append(ret, retV)
	}

	sort.Slice(ret, func(i, j int) bool {
		// Highest-precedence version sorts first
		return ret[i].Version.GreaterThan(ret[j].Version)
	})

	return ret, body.Warnings, nil
}

func (c *RegistryClient) PackageMeta(ctx context.Context, version *ProviderVersion, platform Platform) (*RemotePackageMeta, error) {
	addr := version.Provider
	baseURL, creds, err := c.disco(ctx, addr.Hostname)
	if err != nil {
		return nil, fmt.Errorf("service discovery failed for host %s: %w", addr.Hostname, err)
	}

	endpointPath, err := url.Parse(path.Join(addr.Namespace, addr.Type, "download", platform.OS, platform.Arch))
	if err != nil {
		return nil, fmt.Errorf("failed to construct endpoint path: %w", err)
	}

	endpointURL := baseURL.ResolveReference(endpointPath)
	req, err := http.NewRequestWithContext(ctx, "GET", endpointURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}
	creds.PrepareRequest(req)

	panic("PackageMeta not yet implemented")
}

func (c *RegistryClient) disco(ctx context.Context, hostname svchost.Hostname) (*url.URL, svcauth.HostCredentials, error) {
	baseURL, err := c.services.DiscoverServiceURL(hostname, "providers.v1")
	if err != nil {
		return nil, nil, err
	}
	creds, err := c.services.CredentialsForHost(hostname)
	if err != nil {
		return nil, nil, err
	}
	return baseURL, creds, nil
}
