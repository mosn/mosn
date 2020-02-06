FROM centos:centos7 as builder

ENV CHROOT_MOSN_PREFIX         /ROOT/home/admin/mosn
RUN useradd -ms /bin/bash admin

COPY ./etc/supervisor/supervisord.conf            /ROOT/etc/supervisord.conf
COPY ./etc/supervisor/mosn.conf                   /ROOT/etc/supervisord/conf.d/mosn.conf

COPY ./mosnd $CHROOT_MOSN_PREFIX/bin/mosn

RUN mkdir -p $MOSN_PREFIX/conf \
 && mkdir -p $MOSN_PREFIX/logs

COPY ./configs/mosn_config.json $CHROOT_MOSN_PREFIX/conf/mosn_config.json

RUN chmod +x $CHROOT_MOSN_PREFIX/bin/mosn
RUN chown -R admin:admin /ROOT/home/admin


FROM centos:centos7
ENV TMP_FOLDER      /tmp

RUN yum install -y \
       ssh wget curl perl logrotate make build-essential procps \
       tsar tcpdump mpstat iostat vmstat sysstat \
       python-setuptools; yum clean all

# pip
WORKDIR $TMP_FOLDER
RUN wget https://mirrors.aliyun.com/pypi/packages/69/81/52b68d0a4de760a2f1979b0931ba7889202f302072cc7a0d614211bc7579/pip-18.0.tar.gz#sha256=a0e11645ee37c90b40c46d607070c4fd583e2cd46231b1c06e389c5e814eed76
RUN tar zvxf pip-18.0.tar.gz
WORKDIR $TMP_FOLDER/pip-18.0
RUN python setup.py install

# supervisor
RUN pip install supervisor -i https://mirrors.aliyun.com/pypi/simple
RUN pip install supervisor-stdout -i https://mirrors.aliyun.com/pypi/simple

RUN useradd -ms /bin/bash admin
# copy all in one layer
COPY --from=builder /ROOT /

RUN echo "export MOSN_PREFIX=/home/admin/mosn" >> /etc/bashrc \
 && echo "export PATH=$PATH" >> /etc/bashrc

ENTRYPOINT ["/usr/bin/supervisord" , "-c" , "/etc/supervisord.conf"]
