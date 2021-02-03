FROM centos:centos7

RUN useradd -ms /bin/bash admin

ENV CHROOT_MOSN_PREFIX         /home/admin/mosn

COPY ./envoy $CHROOT_MOSN_PREFIX/bin/envoy

COPY ./golang_extention.so /lib64/

COPY ./conf/envoy.yaml $CHROOT_MOSN_PREFIX/conf/envoy.yaml
COPY ./conf/mosn.json $CHROOT_MOSN_PREFIX/conf/mosn.json
COPY ./conf/service.json $CHROOT_MOSN_PREFIX/discovery/service.json

RUN chmod +x $CHROOT_MOSN_PREFIX/bin/mosn
RUN chown -R admin:admin $CHROOT_MOSN_PREFIX

ENTRYPOINT ["$CHROOT_MOSN_PREFIX/bin/envoy" , "-c" , "$CHROOT_MOSN_PREFIX/conf/envoy.yaml"]
