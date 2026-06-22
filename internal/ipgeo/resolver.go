package ipgeo

import (
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

type Resolver struct {
	db   *geoip2.Reader
	mu   sync.RWMutex
}

type Location struct {
	Country string
	City    string
}

func New(geoIPDBPath string) (*Resolver, error) {
	if geoIPDBPath == "" {
		geoIPDBPath = "GeoLite2-City.mmdb"
	}

	db, err := geoip2.Open(geoIPDBPath)
	if err != nil {
		return nil, err
	}

	return &Resolver{db: db}, nil
}

func (r *Resolver) Resolve(ipStr string) *Location {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &Location{}
	}

	record, err := r.db.City(ip)
	if err != nil {
		return &Location{}
	}

	loc := &Location{
		Country: record.Country.IsoCode,
	}

	if len(record.Subdivisions) > 0 {
		loc.City = record.Subdivisions[0].Names["zh-CN"]
	}
	if record.City.Names["zh-CN"] != "" {
		loc.City = record.City.Names["zh-CN"]
	}

	return loc
}

func (r *Resolver) Close() {
	r.db.Close()
}
