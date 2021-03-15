FROM mosnio/mosn-network-envoy-sdk:1.16.2

ENV CHROOT_MOSN_PREFIX         /home/admin/mosn

COPY ./golang_extention.so /lib64/
COPY ./golang_extention.so /lib/

COPY ./conf/envoy.yaml $CHROOT_MOSN_PREFIX/conf/envoy.yaml
COPY ./conf/mosn.json $CHROOT_MOSN_PREFIX/conf/mosn.json
COPY ./conf/service.json $CHROOT_MOSN_PREFIX/discovery/service.json

RUN chown -R admin:admin $CHROOT_MOSN_PREFIX/conf/ \
 && chown -R admin:admin $CHROOT_MOSN_PREFIX/discovery/ \
 && chmod 777  /lib64/golang_extention.so \
 && chmod 777  /lib/golang_extention.so

ENTRYPOINT ["/home/admin/mosn/bin/envoy" , "-c" , "/home/admin/mosn/conf/envoy.yaml"]
