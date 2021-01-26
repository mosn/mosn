FROM centos:7

RUN yum -y groupinstall 'Development Tools' &&  yum install -y wget zlib* openssl-devel

RUN wget https://www.python.org/ftp/python/3.6.5/Python-3.6.5.tgz && tar -xvf Python-3.6.5.tgz && mkdir -p /usr/local/python3 && \ 
    cd /Python-3.6.5 && \
    ./configure --prefix=/usr/local/python3 --enable-shared --enable-optimizations && \
    make && \
    make install

RUN ln -s /usr/local/python3/bin/python3.6 /usr/bin/python3 && \
    ln -s /usr/local/python3/bin/pip3 /usr/bin/pip3 && \
    cp /Python-3.6.5/libpython3.6m.so.1.0 /usr/lib64 && \
    cp /Python-3.6.5/libpython3.6m.so /usr/lib64 && \
    cp /Python-3.6.5/libpython3.so /usr/lib64 && \
    ls /Python-3.6.5/*.so* && \
    rm -f Python-3.6.5.tgz && rm -rf /Python-3.6.5 && \
    pip3 install --upgrade pip && pip3 install PyInstaller grpcio-health-checking protobuf