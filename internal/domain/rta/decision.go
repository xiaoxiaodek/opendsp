package rta

// Decision represents an advertiser's RTA response for a specific user.
type Decision struct {
	Allow  bool
	Reason string
}

// AllowAll returns a decision that allows all candidates (pass-through skeleton).
func AllowAll() Decision {
	return Decision{Allow: true, Reason: "rta_disabled"}
}

// DenyAll returns a decision that denies all candidates.
func DenyAll(reason string) Decision {
	return Decision{Allow: false, Reason: reason}
}
