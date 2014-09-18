FROM registry.edmodo.io/base

ADD start.sh /home/deployer/start.sh
ADD docker-proxy /home/deployer/docker-proxy.bin
ENTRYPOINT /home/deployer/start.sh
