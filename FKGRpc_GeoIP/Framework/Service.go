//---------------------------------------------
package framework

//---------------------------------------------
import (
	PROTO "FKGoServer/FKGRpc_GeoIP/Proto"
	ERRORS "errors"
	NET "net"
	OS "os"
	STRINGS "strings"

	MAXMINDDB "github.com/oschwald/maxminddb-golang"

	LOG "github.com/Sirupsen/logrus"
	CONTEXT "golang.org/x/net/context"
)

//---------------------------------------------
const (
	SERVICE = "[GEOIP]"
)

//---------------------------------------------
var (
	ERROR_CANNOT_QUERY_IP = ERRORS.New("cannot query ip")
)

//---------------------------------------------
// read the following fields only
type City struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`

	Country struct {
		GeoNameID uint   `maxminddb:"geoname_id"`
		IsoCode   string `maxminddb:"iso_code"`
	} `maxminddb:"country"`

	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

//---------------------------------------------
type Server struct {
	mmdb *MAXMINDDB.Reader
}

//---------------------------------------------
func (s *Server) Func_Init() {
	// 载入IP表
	LOG.Debug("Loading GEOIP City...")
	reader, err := MAXMINDDB.Open(s.data_path())
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}

	s.mmdb = reader
	LOG.Debug("GEOIP City Load Complete.")
}

//---------------------------------------------
// get correct data path from GOPATH
func (s *Server) data_path() (path string) {
	paths := STRINGS.Split(OS.Getenv("GOPATH"), ";")
	for k := range paths {
		path = paths[k] + "/src/FKGoServer/FKGRpc_GeoOP/Res/GeoIP2-City.mmdb"
		_, err := OS.Lstat(path)
		if err == nil {
			return path
		}
	}
	return
}

//---------------------------------------------
func (s *Server) query(ip NET.IP) *City {
	city := &City{}
	err := s.mmdb.Lookup(ip, city)
	if err != nil {
		LOG.Error(err)
		return nil
	}

	return city
}

//---------------------------------------------
// 查询IP所属国家
func (s *Server) QueryCountry(ctx CONTEXT.Context, in *PROTO.GeoIP_IP) (*PROTO.GeoIP_Name, error) {
	ip := NET.ParseIP(in.Ip)
	if city := s.query(ip); city != nil {
		return &PROTO.GeoIP_Name{Name: city.Country.IsoCode}, nil
	}
	return nil, ERROR_CANNOT_QUERY_IP
}

//---------------------------------------------
// 查询IP所属城市
func (s *Server) QueryCity(ctx CONTEXT.Context, in *PROTO.GeoIP_IP) (*PROTO.GeoIP_Name, error) {
	ip := NET.ParseIP(in.Ip)
	if city := s.query(ip); city != nil {
		return &PROTO.GeoIP_Name{Name: city.City.Names["en"]}, nil
	}
	return nil, ERROR_CANNOT_QUERY_IP
}

//---------------------------------------------
// 查询IP所属地区(省)
func (s *Server) QuerySubdivision(ctx CONTEXT.Context, in *PROTO.GeoIP_IP) (*PROTO.GeoIP_Name, error) {
	ip := NET.ParseIP(in.Ip)
	if city := s.query(ip); city != nil {
		if len(city.Subdivisions) > 0 {
			return &PROTO.GeoIP_Name{Name: city.Subdivisions[0].Names["en"]}, nil
		}
		return nil, ERROR_CANNOT_QUERY_IP
	}
	return nil, ERROR_CANNOT_QUERY_IP
}

//---------------------------------------------
