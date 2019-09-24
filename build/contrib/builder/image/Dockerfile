FROM centos:centos7
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

# pip
WORKDIR $TMP_FOLDER
RUN wget https://mirrors.aliyun.com/pypi/packages/69/81/52b68d0a4de760a2f1979b0931ba7889202f302072cc7a0d614211bc7579/pip-18.0.tar.gz#sha256=a0e11645ee37c90b40c46d607070c4fd583e2cd46231b1c06e389c5e814eed76
RUN tar zvxf pip-18.0.tar.gz
WORKDIR $TMP_FOLDER/pip-18.0
RUN python setup.py install

# supervisor
RUN pip install supervisor -i https://mirrors.aliyun.com/pypi/simple
RUN pip install supervisor-stdout -i https://mirrors.aliyun.com/pypi/simple

RUN echo "export MOSN_PREFIX=/home/admin/mosn" >> /etc/bashrc;
RUN echo "export PATH=$PATH" >> /etc/bashrc

COPY ./etc/supervisor/supervisord.conf            /etc/supervisord.conf
COPY ./etc/supervisor/mosn.conf                   /etc/supervisord/conf.d/mosn.conf
COPY ./etc/mosn/mosn.logrotate                    /etc/logrotate.d/mosn
RUN mv /etc/cron.daily/logrotate                  /etc/cron.hourly/logrotate

EXPOSE 22 80

ADD ./mosnd $MOSN_PREFIX/sbin

ENTRYPOINT ["/usr/bin/supervisord" , "-c" , "/etc/supervisord.conf"]
