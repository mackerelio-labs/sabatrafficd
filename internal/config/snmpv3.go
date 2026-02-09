package config

import (
	"slices"
	"strings"

	"github.com/gosnmp/gosnmp"
)

type yamlCollectorConfigSNMPv3 struct {
	SecLevel string `yaml:"security"` // auth priv noauth

	UserName                 string `yaml:"username"`
	AuthenticationProtocol   string `yaml:"auth-protocol"`
	AuthenticationPassphrase string `yaml:"auth-password"`
	PrivacyProtocol          string `yaml:"priv-protocol"`
	PrivacyPassphrase        string `yaml:"priv-password"`
}

type collectorSNMPConfigV3 struct {
	secLevel string

	usename                  string
	authenticationProtocol   string
	authenticationPassphrase string
	privacyProtocol          string
	privacyPassphrase        string
}

const (
	secLevelNoAuth = "noauth"
	secLevelAuth   = "auth"
	secLevelPriv   = "priv"
)

func parseSeurity(v string) bool {
	return slices.Contains([]string{secLevelNoAuth, secLevelAuth, secLevelPriv}, v)
}

func (c *collectorSNMPConfigV3) MsgFlags() gosnmp.SnmpV3MsgFlags {
	switch strings.ToLower(c.secLevel) {
	case secLevelNoAuth:
		return gosnmp.NoAuthNoPriv
	case secLevelAuth:
		return gosnmp.AuthNoPriv
	case secLevelPriv:
		return gosnmp.AuthPriv
	}
	panic("norechable")
}

const (
	authProtoNoAuth = "noauth"
	authProtoMD5    = "md5"
	authProtoSHA    = "sha"
	authProtoSHA224 = "sha224"
	authProtoSHA256 = "sha256"
	authProtoSHA384 = "sha384"
	authProtoSHA512 = "sha512"
)

func parseAuthenticationProtocol(v string) bool {
	return slices.Contains([]string{
		authProtoNoAuth,
		authProtoMD5,
		authProtoSHA,
		authProtoSHA224,
		authProtoSHA256,
		authProtoSHA384,
		authProtoSHA512,
	}, v)
}

const (
	privNoPriv = "nopriv"
	privDES    = "des"
	privAES    = "aes"
	privAES192 = "aes192"
	privAES256 = "aes256"
	// AES192C = ""
	// AES256C = ""
)

func parsePrivacyProtocol(v string) bool {
	return slices.Contains([]string{
		privNoPriv,
		privDES,
		privAES,
		privAES192,
		privAES256,
	}, v)
}

func (c *collectorSNMPConfigV3) SecurityParameters() *gosnmp.UsmSecurityParameters {
	var authProtocol gosnmp.SnmpV3AuthProtocol
	switch c.authenticationProtocol {
	case authProtoNoAuth:
		authProtocol = gosnmp.NoAuth
	case authProtoMD5:
		authProtocol = gosnmp.MD5
	case authProtoSHA:
		authProtocol = gosnmp.SHA
	case authProtoSHA224:
		authProtocol = gosnmp.SHA224
	case authProtoSHA256:
		authProtocol = gosnmp.SHA256
	case authProtoSHA384:
		authProtocol = gosnmp.SHA384
	case authProtoSHA512:
		authProtocol = gosnmp.SHA512
	}

	var privacyProtocol gosnmp.SnmpV3PrivProtocol
	switch c.privacyProtocol {
	case privNoPriv:
		privacyProtocol = gosnmp.NoPriv
	case privDES:
		privacyProtocol = gosnmp.DES
	case privAES:
		privacyProtocol = gosnmp.AES
	case privAES192:
		privacyProtocol = gosnmp.AES192
	case privAES256:
		privacyProtocol = gosnmp.AES256
	}

	return &gosnmp.UsmSecurityParameters{
		UserName:                 c.usename,
		AuthenticationProtocol:   authProtocol,
		AuthenticationPassphrase: c.authenticationPassphrase,
		PrivacyProtocol:          privacyProtocol,
		PrivacyPassphrase:        c.privacyPassphrase,
	}
}
