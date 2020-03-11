package tls

import (
	gox509 "crypto/x509"

	"mosn.io/mosn/pkg/mtls/crypto/x509"
)

func transferKeyUseage(b []x509.ExtKeyUsage) []gox509.ExtKeyUsage {
	res := make([]gox509.ExtKeyUsage, len(b))
	for _, k := range b {
		res = append(res, gox509.ExtKeyUsage(k))
	}
	return res
}

func TransferBabasslX509(babaX509 *x509.Certificate) *gox509.Certificate {
	return &gox509.Certificate{
		Raw:                         babaX509.Raw,
		RawTBSCertificate:           babaX509.RawTBSCertificate,
		RawSubjectPublicKeyInfo:     babaX509.RawSubjectPublicKeyInfo,
		RawSubject:                  babaX509.RawSubject,
		RawIssuer:                   babaX509.RawIssuer,
		Signature:                   babaX509.Signature,
		SignatureAlgorithm:          gox509.SignatureAlgorithm(babaX509.SignatureAlgorithm),
		PublicKeyAlgorithm:          gox509.PublicKeyAlgorithm(babaX509.PublicKeyAlgorithm),
		PublicKey:                   babaX509.PublicKey,
		Version:                     babaX509.Version,
		SerialNumber:                babaX509.SerialNumber,
		Issuer:                      babaX509.Issuer,
		Subject:                     babaX509.Subject,
		NotBefore:                   babaX509.NotBefore,
		NotAfter:                    babaX509.NotAfter,
		KeyUsage:                    gox509.KeyUsage(babaX509.KeyUsage),
		Extensions:                  babaX509.Extensions,
		ExtraExtensions:             babaX509.ExtraExtensions,
		UnhandledCriticalExtensions: babaX509.UnhandledCriticalExtensions,
		ExtKeyUsage:                 transferKeyUseage(babaX509.ExtKeyUsage),
		UnknownExtKeyUsage:          babaX509.UnknownExtKeyUsage,
		BasicConstraintsValid:       babaX509.BasicConstraintsValid,
		IsCA:                        babaX509.IsCA,
		MaxPathLen:                  babaX509.MaxPathLen,
		MaxPathLenZero:              babaX509.MaxPathLenZero,
		SubjectKeyId:                babaX509.SubjectKeyId,
		AuthorityKeyId:              babaX509.AuthorityKeyId,
		OCSPServer:                  babaX509.OCSPServer,
		IssuingCertificateURL:       babaX509.IssuingCertificateURL,
		DNSNames:                    babaX509.DNSNames,
		EmailAddresses:              babaX509.EmailAddresses,
		IPAddresses:                 babaX509.IPAddresses,
		URIs:                        babaX509.URIs,
		PermittedDNSDomainsCritical: babaX509.PermittedDNSDomainsCritical,
		PermittedDNSDomains:         babaX509.PermittedDNSDomains,
		ExcludedDNSDomains:          babaX509.ExcludedDNSDomains,
		PermittedIPRanges:           babaX509.PermittedIPRanges,
		ExcludedIPRanges:            babaX509.ExcludedIPRanges,
		PermittedEmailAddresses:     babaX509.PermittedEmailAddresses,
		ExcludedEmailAddresses:      babaX509.ExcludedEmailAddresses,
		PermittedURIDomains:         babaX509.PermittedURIDomains,
		ExcludedURIDomains:          babaX509.ExcludedURIDomains,
		CRLDistributionPoints:       babaX509.CRLDistributionPoints,
		PolicyIdentifiers:           babaX509.PolicyIdentifiers,
	}
}

func TransferSliceX509toGoX509(certs []*x509.Certificate) []*gox509.Certificate {
	res := make([]*gox509.Certificate, len(certs), len(certs))
	for i, cert := range certs {
		res[i] = TransferBabasslX509(cert)
	}
	return res
}

func TransferSliceArrX509toGoX509(certsArr [][]*x509.Certificate) [][]*gox509.Certificate {
	res := make([][]*gox509.Certificate, len(certsArr), len(certsArr))
	for i, certs := range certsArr {
		res[i] = TransferSliceX509toGoX509(certs)
	}
	return res
}
