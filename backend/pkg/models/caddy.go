package models

// CaddyConfig represents the Caddy JSON configuration structure
type CaddyConfig struct {
	Apps CaddyApps `json:"apps"`
}

type CaddyApps struct {
	HTTP CaddyHTTP `json:"http"`
	TLS  *CaddyTLS `json:"tls,omitempty"`
}

type CaddyHTTP struct {
	Servers map[string]CaddyServer `json:"servers"`
}

type CaddyServer struct {
	Listen         []string             `json:"listen"`
	Routes         []CaddyRoute         `json:"routes"`
	AutomaticHTTPS *CaddyAutomaticHTTPS `json:"automatic_https,omitempty"`
	TLSPolicies    []CaddyTLSPolicy     `json:"tls_policies,omitempty"`
}

type CaddyAutomaticHTTPS struct {
	Disable bool `json:"disable"`
}

type CaddyRoute struct {
	ID     string         `json:"@id,omitempty"`
	Match  []CaddyMatch   `json:"match"`
	Handle []CaddyHandler `json:"handle"`
}

type CaddyMatch struct {
	Host []string `json:"host"`
}

type CaddyHandler struct {
	Handler   string                     `json:"handler"`
	Upstreams []CaddyUpstream            `json:"upstreams,omitempty"`
	Transport *CaddyTransport            `json:"transport,omitempty"`
	Headers   *CaddyHeaders              `json:"headers,omitempty"`
	Providers map[string]CaddyAuthProvider `json:"providers,omitempty"` // For basic auth - must be a map
}

type CaddyAuthProvider struct {
	Accounts []CaddyAccount `json:"accounts"`
}

type CaddyAccount struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CaddyHeaders struct {
	Request *CaddyHeadersRequest `json:"request,omitempty"`
}

type CaddyHeadersRequest struct {
	Set map[string][]string `json:"set,omitempty"`
}

type CaddyTransport struct {
	Protocol string    `json:"protocol"`
	TLS      *struct{} `json:"tls,omitempty"`
}

type CaddyUpstream struct {
	Dial string `json:"dial"`
}

// TLS and ACME structures for DNS challenge support

type CaddyTLS struct {
	CertificateAuthorities map[string]CaddyCA `json:"certificate_authorities,omitempty"`
}

type CaddyCA struct {
	Module     string          `json:"module"`
	Challenges CaddyChallenges `json:"challenges,omitempty"`
}

type CaddyChallenges struct {
	DNS *CaddyDNSChallenge `json:"dns,omitempty"`
}

type CaddyDNSChallenge struct {
	Provider CaddyDNSProvider `json:"provider"`
}

type CaddyDNSProvider struct {
	Name      string `json:"name"`
	APIToken  string `json:"api_token,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
	Token     string `json:"token,omitempty"`
	Email     string `json:"email,omitempty"`
}

type CaddyTLSPolicy struct {
	Match   *CaddyTLSMatch `json:"match,omitempty"`
	Issuers []CaddyIssuer  `json:"issuers,omitempty"`
}

type CaddyTLSMatch struct {
	SNI []string `json:"sni,omitempty"`
}

type CaddyIssuer struct {
	Module     string          `json:"module"`
	Challenges CaddyChallenges `json:"challenges,omitempty"`
}
