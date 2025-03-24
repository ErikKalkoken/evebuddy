package app

type RoutePreference string

const (
	RouteShortest RoutePreference = "shortest"
	RouteSecure   RoutePreference = "secure"
	RouteInsecure RoutePreference = "insecure"
)

func (x RoutePreference) String() string {
	return string(x)
}

func RoutePreferences() []RoutePreference {
	return []RoutePreference{RouteShortest, RouteSecure, RouteInsecure}
}
