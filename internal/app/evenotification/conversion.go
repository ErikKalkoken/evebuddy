package evenotification

import "time"

// fromLDAPTime converts an ldap time to golang time
func fromLDAPTime(ldap_dt int64) time.Time {
	return time.Unix((ldap_dt/10000000)-11644473600, 0).UTC()
}

// fromLDAPDuration converts an ldap duration to golang duration
func fromLDAPDuration(ldap_td int64) time.Duration {
	return time.Duration(ldap_td/10) * time.Microsecond
}
