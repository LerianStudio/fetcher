package setup

// PortProvider is implemented by containers that have a port.
type PortProvider interface {
	GetPort() string
}

// HostProvider is implemented by containers that have a host.
type HostProvider interface {
	GetHost() string
}

// URIProvider is implemented by containers that have a connection URI.
type URIProvider interface {
	GetURI() string
}

// GetPort returns the port from a container, or empty string if nil.
// This replaces the 8 duplicated getXXXPort functions.
func GetPort(container any) string {
	if container == nil {
		return ""
	}

	if p, ok := container.(PortProvider); ok {
		return p.GetPort()
	}

	return ""
}

// GetHost returns the host from a container, or empty string if nil.
func GetHost(container any) string {
	if container == nil {
		return ""
	}

	if h, ok := container.(HostProvider); ok {
		return h.GetHost()
	}

	return ""
}

// GetURI returns the URI from a container, or empty string if nil.
func GetURI(container any) string {
	if container == nil {
		return ""
	}

	if u, ok := container.(URIProvider); ok {
		return u.GetURI()
	}

	return ""
}
