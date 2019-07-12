FROM centos:centos6
MAINTAINER  xiaodong.dxd@alibaba-inc.com

ENV TMP_FOLDER      /tmp
ENV GIT_USER        alipay
ENV MOSN_PREFIX     /home/admin/mosn
ENV PATH            $MOSN_PREFIX/sbin:$PATH

RUN mkdir -p $MOSN_PREFIX/conf
RUN mkdir -p $MOSN_PREFIX/logs

RUN yum install -y \
       ssh wget curl perl logrotate make build-essential procps \
       tsar tcpdump mpstat iostat vmstat sysstat \
       python-setuptools; yum clean all

# 规避crond在docker环境下运行的问题
RUN echo -e "sleep 10;/bin/touch /var/spool/cron/a;/bin/rm -f /var/spool/cron/a;service crond restart" > /opt/fix-cron.sh

COPY ./etc/script/entrypoint.sh     /usr/local/bin/entrypoint.sh

# supervisor
RUN easy_install supervisor==3.4.0

# supervisor-stdout
WORKDIR $TMP_FOLDER
COPY ./etc/libs/pip/pip-1.5.5.tar.gz ./pip-1.5.5.tar.gz
RUN tar zvxf pip-1.5.5.tar.gz
WORKDIR $TMP_FOLDER/pip-1.5.5
RUN python setup.py install
RUN pip install supervisor-stdout

RUN echo "export MOSN_PREFIX=/home/admin/mosn" >> /etc/bashrc;
RUN echo "export PATH=$PATH" >> /etc/bashrc

COPY ./etc/supervisor/supervisord.conf            /etc/supervisord.conf
COPY ./etc/supervisor/mosn.conf                   /etc/supervisord/conf.d/mosn.conf
COPY ./etc/mosn/mosn.logrotate                    /etc/logrotate.d/mosn
RUN mv /etc/cron.daily/logrotate                  /etc/cron.hourly/logrotate

EXPOSE 22 80

ADD ./mosnd $MOSN_PREFIX/sbin

ENTRYPOINT ["/usr/bin/supervisord" , "-c" , "/etc/supervisord.conf"]
