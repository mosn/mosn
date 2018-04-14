FROM acs-reg.alipay.com/alipay/alios6u2:1.0.1-4165050
MAINTAINER  xiaodong.dxd@alibaba-inc.com

ADD ./mosnd /usr/bin

COPY ./etc/script/entrypoint.sh     /usr/local/bin/entrypoint.sh

EXPOSE 22 80

ENTRYPOINT ["sh", "/usr/local/bin/entrypoint.sh"]
CMD ["--help"]