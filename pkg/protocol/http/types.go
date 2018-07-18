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

package http

type Code uint32

const (
	Continue Code = 100
	OK            = 200

	Created                     = 201
	Accepted                    = 202
	NonAuthoritativeInformation = 203
	NoContent                   = 204
	ResetContent                = 205
	PartialContent              = 206
	MultiStatus                 = 207
	AlreadyReported             = 208
	IMUsed                      = 226

	MultipleChoices   = 300
	MovedPermanently  = 301
	Found             = 302
	SeeOther          = 303
	NotModified       = 304
	UseProxy          = 305
	TemporaryRedirect = 307
	PermanentRedirect = 308

	BadRequest                  = 400
	Unauthorized                = 401
	PaymentRequired             = 402
	Forbidden                   = 403
	NotFound                    = 404
	MethodNotAllowed            = 405
	NotAcceptable               = 406
	ProxyAuthenticationRequired = 407
	RequestTimeout              = 408
	Conflict                    = 409
	Gone                        = 410
	LengthRequired              = 411
	PreconditionFailed          = 412
	PayloadTooLarge             = 413
	URITooLong                  = 414
	UnsupportedMediaType        = 415
	RangeNotSatisfiable         = 416
	ExpectationFailed           = 417
	MisdirectedRequest          = 421
	UnprocessableEntity         = 422
	Locked                      = 423
	FailedDependency            = 424
	UpgradeRequired             = 426
	PreconditionRequired        = 428
	TooManyRequests             = 429
	RequestHeaderFieldsTooLarge = 431

	InternalServerError           = 500
	NotImplemented                = 501
	BadGateway                    = 502
	ServiceUnavailable            = 503
	GatewayTimeout                = 504
	HTTPVersionNotSupported       = 505
	VariantAlsoNegotiates         = 506
	InsufficientStorage           = 507
	LoopDetected                  = 508
	NotExtended                   = 510
	NetworkAuthenticationRequired = 511
)
