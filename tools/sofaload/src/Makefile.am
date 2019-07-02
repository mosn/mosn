# nghttp2 - HTTP/2 C Library

# Copyright (c) 2012 Tatsuhiro Tsujikawa

# Permission is hereby granted, free of charge, to any person obtaining
# a copy of this software and associated documentation files (the
# "Software"), to deal in the Software without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so, subject to
# the following conditions:

# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.

# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
# LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
# OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
# WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
SUBDIRS = includes

EXTRA_DIST = CMakeLists.txt

bin_PROGRAMS =
check_PROGRAMS =
TESTS =

AM_CFLAGS = $(WARNCFLAGS)
AM_CXXFLAGS = $(WARNCXXFLAGS) $(CXX1XCXXFLAGS)
AM_CPPFLAGS = \
	-DPKGDATADIR='"$(pkgdatadir)"' \
	-I$(top_srcdir)/lib/includes \
	-I$(top_builddir)/lib/includes \
	-I$(top_srcdir)/lib \
	-I$(top_srcdir)/src/includes \
	-I$(top_srcdir)/third-party \
	-I$(top_srcdir)/third-party/llhttp/include \
	@LIBXML2_CFLAGS@ \
	@LIBEV_CFLAGS@ \
	@OPENSSL_CFLAGS@ \
	@LIBCARES_CFLAGS@ \
	@JANSSON_CFLAGS@ \
	@ZLIB_CFLAGS@ \
	@DEFS@

LDADD = $(top_builddir)/lib/libnghttp2.la \
	$(top_builddir)/third-party/liburl-parser.la \
	$(top_builddir)/third-party/libllhttp.la \
	@JEMALLOC_LIBS@ \
	@LIBXML2_LIBS@ \
	@LIBEV_LIBS@ \
	@OPENSSL_LIBS@ \
	@LIBCARES_LIBS@ \
	@SYSTEMD_LIBS@ \
	@JANSSON_LIBS@ \
	@ZLIB_LIBS@ \
	@APPLDFLAGS@

if ENABLE_APP

bin_PROGRAMS += sofaload

sofaload_SOURCES = util.cc util.h \
	http2.cc http2.h h2load.cc h2load.h \
	timegm.c timegm.h \
	tls.cc tls.h \
	h2load_session.h \
	h2load_http2_session.cc h2load_http2_session.h \
	h2load_http1_session.cc h2load_http1_session.h \
	h2load_sofarpc_session.cc h2load_sofarpc_session.h

endif # ENABLE_APP
