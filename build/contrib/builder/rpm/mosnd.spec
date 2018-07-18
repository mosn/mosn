Name:       %{AFENP_NAME}
Version:    %{AFENP_VERSION}
Release:    %{AFENP_RELEASE}
Source:     %{name}-%{version}.tar.gz
License:    MIT
Group:      Tools/Docker
Packager:   ant-frontend <o_dept_55122@list.alibaba-inc.com>
Vendor:     Alipay.com
Summary:    alipay sofa-mosn

%define _homedir /home/admin/mosn

%description
Mosn is a net stub, used as a mesh sidecar

%{AFENP_GIT_NOTES}

%prep
rm -rf %{buildroot}

%setup -q

%build

%install
mkdir -p $RPM_BUILD_ROOT/%{_homedir}/bin
mkdir -p $RPM_BUILD_ROOT/%{_homedir}/conf
install -m 755 mosnd $RPM_BUILD_ROOT/%{_homedir}/bin
install -m 666 mosn_config.json $RPM_BUILD_ROOT/%{_homedir}/conf
mkdir -p $RPM_BUILD_ROOT/etc/init.d
mkdir -p $RPM_BUILD_ROOT/etc/logrotate.d
install -m 755 mosnd.service $RPM_BUILD_ROOT/etc/init.d/mosnd
install -m 755 mosnd.logrotate $RPM_BUILD_ROOT/etc/logrotate.d/mosnd
ls -al $RPM_BUILD_ROOT/%{_homedir}/bin
ls -al $RPM_BUILD_ROOT/%{_homedir}/conf

%preun

%post
cp /etc/cron.daily/logrotate   /etc/cron.hourly/

%files
%{_homedir}/bin/mosnd
%config(noreplace) %{_homedir}/conf/mosn_config.json
/etc/init.d/mosnd
/etc/logrotate.d/mosnd

%clean
rm -rf $RPM_BUILD_ROOT
