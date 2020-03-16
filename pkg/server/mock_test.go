/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package server

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type mockClusterManager struct {
	types.ClusterManager
}

type mockClusterManagerFilter struct {
	cccb types.ClusterConfigFactoryCb
	chcb types.ClusterHostFactoryCb
}

func (cmf *mockClusterManagerFilter) OnCreated(cccb types.ClusterConfigFactoryCb, chcb types.ClusterHostFactoryCb) {
	cmf.cccb = cccb
	cmf.chcb = chcb
}

type mockNetworkFilter struct{}

func (nf *mockNetworkFilter) OnData(buffer buffer.IoBuffer) api.FilterStatus {
	return api.Stop
}
func (nf *mockNetworkFilter) OnNewConnection() api.FilterStatus {
	return api.Continue
}
func (nf *mockNetworkFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {}

type mockNetworkFilterFactory struct{}

func (ff *mockNetworkFilterFactory) CreateFilterChain(context context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	callbacks.AddReadFilter(&mockNetworkFilter{})
}

func CreateMockFilerFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	return &mockNetworkFilterFactory{}, nil
}

type mockStreamFilterFactory struct{}

func (ff *mockStreamFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
}

func CreateMockStreamFilterFactory(conf map[string]interface{}) (api.StreamFilterChainFactory, error) {
	return &mockStreamFilterFactory{}, nil
}

func init() {
	api.RegisterNetwork("mock_network", CreateMockFilerFactory)
	api.RegisterNetwork("mock_network2", CreateMockFilerFactory)
	api.RegisterStream("mock_stream", CreateMockStreamFilterFactory)
	api.RegisterStream("mock_stream2", CreateMockStreamFilterFactory)

}

const mockCAPEM = `-----BEGIN CERTIFICATE-----
MIIC4jCCAcqgAwIBAgIQdme9dUKPZVTgfIVpy+fmmjANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMB4XDTE5MDIyMDA3NTMyOFoXDTIwMDIyMDA3NTMy
OFowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAK/6yPpumKMxdVJU1y4ssNaqCp1NWJogss7cnF6YJHxYI64UXPnbAcSw
oWxnIKqjtY6c+6tT2G3HRrHB+gE3jv7aVcUf0wyXSWMoUfzGKoigPbPHSdSS3Ri8
+snJz3+tnJkmL4Mso0c8MBcdX8+65FTDTxk9wV0ZWjNty0kf+udyJb8gEaHOTsjS
XypqFc+GIezjpQICqWNdFHILf+F/WgVT4iHfxUlZBlXQxA945dS6v6pEcKF07Tqv
UEVSVO5QC42l/8ahCmwSbzWckX12dVvKbEKRsyDUxP/BzZBTd6fTcD6KQgdOdofz
LkNxegY0UP/d9KzvKdbNATo6TK34OlUCAwEAAaM0MDIwDgYDVR0PAQH/BAQDAgKk
MA8GA1UdEwEB/wQFMAMBAf8wDwYDVR0RBAgwBocEfwAAATANBgkqhkiG9w0BAQsF
AAOCAQEAYMi10S2KT5EH9o0GxmbSWLArarHFtBME2HdYip1p8+uW+J5KNqk97yEa
Naqy74bekBlD7rWmUnhJM6hCHvYYzoqGaiFVp8LbRtH4vCNG1slL/OMyleGyLJ/Z
pYK19tjyMk2HLwz9LMVVqDohk1hCpWIene1+hhNhVSqv2VAI3q176T4vNrXtcgXr
xO8EIu/BnMMLPFSTlSAiJ5VHZITRfHSbabrr65onOR3JxoO65+UnIT05yHI1RDsn
aJ16S3TSuSNRabZ+ji9BNhIP3DzPMIIGfmrev3wcgZsJB/ZmjRMIbALiwEoW1z8h
BZljtrz0tsrbWvYDKMvVOeDGghy/KQ==
-----END CERTIFICATE-----`

const mockCertPEM = `-----BEGIN CERTIFICATE-----
MIIC/jCCAeagAwIBAgIQdme9dUKPZVTgfIVpy+fmmjANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMB4XDTE5MDIyMDA3NTMyOFoXDTIwMDIyMDA3NTMy
OFowEjEQMA4GA1UEChMHQWNtZSBDbzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCC
AQoCggEBAK/6yPpumKMxdVJU1y4ssNaqCp1NWJogss7cnF6YJHxYI64UXPnbAcSw
oWxnIKqjtY6c+6tT2G3HRrHB+gE3jv7aVcUf0wyXSWMoUfzGKoigPbPHSdSS3Ri8
+snJz3+tnJkmL4Mso0c8MBcdX8+65FTDTxk9wV0ZWjNty0kf+udyJb8gEaHOTsjS
XypqFc+GIezjpQICqWNdFHILf+F/WgVT4iHfxUlZBlXQxA945dS6v6pEcKF07Tqv
UEVSVO5QC42l/8ahCmwSbzWckX12dVvKbEKRsyDUxP/BzZBTd6fTcD6KQgdOdofz
LkNxegY0UP/d9KzvKdbNATo6TK34OlUCAwEAAaNQME4wDgYDVR0PAQH/BAQDAgWg
MB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8G
A1UdEQQIMAaHBH8AAAEwDQYJKoZIhvcNAQELBQADggEBAJkBEahCiyNAAAsoDgJy
oYVr9W4FgrX1VMM5DFG6jX4aw4oDC91ZEqNK6VYkO1QIsSaLELBZZ+oB6kgM8N/4
VgQnOJup8s0MAktJjudxQO81BYCnONKxktOUblj+kl4qOy6W8BH2yQ5gZO25ZTE1
4hKuGRJIK2WGvDYye5SHmjnGUDmM3dVTSmIc8UmXP/xDtDZwHxj2BarkrHwf9k5/
Q0WtzclIj4DUNh19FbSVq+zFJnPHDdhoo9zgkW7vDSutd41F3euCLtKwng02CLkw
g+H3xKcfGv2gNrL7u8ej5l4dfWtWRsmLAG3FEUqsRBfG1PMX3QmB2AUpArfJE7Pu
nCQ=
-----END CERTIFICATE-----`

const mockKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAr/rI+m6YozF1UlTXLiyw1qoKnU1YmiCyztycXpgkfFgjrhRc
+dsBxLChbGcgqqO1jpz7q1PYbcdGscH6ATeO/tpVxR/TDJdJYyhR/MYqiKA9s8dJ
1JLdGLz6ycnPf62cmSYvgyyjRzwwFx1fz7rkVMNPGT3BXRlaM23LSR/653IlvyAR
oc5OyNJfKmoVz4Yh7OOlAgKpY10Ucgt/4X9aBVPiId/FSVkGVdDED3jl1Lq/qkRw
oXTtOq9QRVJU7lALjaX/xqEKbBJvNZyRfXZ1W8psQpGzINTE/8HNkFN3p9NwPopC
B052h/MuQ3F6BjRQ/930rO8p1s0BOjpMrfg6VQIDAQABAoIBAQCGHAB9mTsJYu+d
xroVnklFzmA4cHFNRA4AR2+DRz7G5ASM7UfNwXEfi9v42L60S/5YqJnCfys4vdzK
KqFzu/tljM5AY3ha6BAtWNTiZcKUTEm5b+576VBFQf99OCbBjnUA4XDj7migKOYd
N22EyVCoqA7nlYB+iouLFekN2SlEpxx5Nj8QLqHUReV9vpU+vBfK/OdkLoaZOECS
ffMWSRxYrS01XW0hp/MTtpGwayt1mNvj36+Gwn3Z5p967uXy7qK0bxn8IJYHbe3F
Tv3SRSLf2do3CEYyanpEv7J3QMtwZeanHwumMotvrP8gQpiK55QldEzvpW9JoHa7
gU+nOncBAoGBAOPojo9yQLzzkZFAmZNMcQn7DuVD7iEGw1AFtr8yhmVbjNcuXJoq
isaCxnwJRvkBmf9U+tpMDoxOT1IOjGUCfVuTncmpd3Qydxt+xq2/kf1scptygqQl
bjQ06N64BmvMVys31tS+EFhRSJR8esTzkwmqI3bB+tUdQi2C8qn7DHtZAoGBAMWr
qYlerj23CH6B1/UjKaUh/SssJpoeXAtO7/j2gZ1vNpb5Mt20soFKxN8i38+04mon
yyYGvX4Y1+2ju/fd8ILFvYG2WPAHJi/7P/81ktcgCKnkwQnXrlfMUEOmi0eOTi8P
8GN1dM/CAJikogaleLqJJcrInsxNiLFfpmJRHGNdAoGAMLwD8AygZ0c2M3c639KS
wW2cC85w10MY9L2kDFKDhp0DCuhxCM5cCoLgapmZQZnkEkNbuN5Wpg4AzC0sPFVB
9Rklvn+seX5pFcoQNgsm7qgIAdGEuhD+9c7ylN2JEfgKE8XG/Ir/98K54HaV0hO7
t29YUga82mF9SzobJdn3G1ECgYBqD31b861R989a8ZhKM5+4ts/8RihAMWH5v1UL
JFjPfEiyIOumAbp1nQSdJT0pWUjS5J8fvCYYboQNQfktOaw+vpK78nct8ugOfqUL
7lbnjoyXe+IHwe4NtdarNcUtk7FnlwnIk9ElWFaxkERPhKGOlN/uOk7aGA/r/AJu
Zk7xEQKBgDh4B7wTVi6eKRDYJpgWiDgT5uV0gu60mJ0CHYmjUtX9RGBsGS85zJmx
2JMjNOmrivQ3dvL/rNKIx0ULx+iTQr3Y6B8A2xCme3yr665arvE0HOFG0hLVE1a0
wD08P7/0q7yk5M4dDUwerGaTIRKK8RRFy1Ak9kU6EtHsbUKNo9s5
-----END RSA PRIVATE KEY-----`
